/*
Copyright © 2026 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	agentproxy "github.com/DopplerHQ/agent-proxy"
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/proxy"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Run a credential-injecting proxy for AI agents (experimental)",
	Args:  cobra.NoArgs,
}

var proxyStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the agent proxy",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		engineName, _ := cmd.Flags().GetString("engine")
		address, _ := cmd.Flags().GetString("address")

		// Resolve the CLI's auth + scope the same way `doppler run` does.
		localConfig := configuration.LocalConfig(cmd)
		utils.RequireValue("token", localConfig.Token.Value)

		// A config-scoped service token (dp.st.) carries its own project/config.
		// Otherwise we need a selected project + config — guide the user to
		// `doppler setup` instead of failing later with a raw API error.
		tokenIsConfigScoped := strings.HasPrefix(localConfig.Token.Value, "dp.st.")
		if !tokenIsConfigScoped && (localConfig.EnclaveProject.Value == "" || localConfig.EnclaveConfig.Value == "") {
			utils.HandleError(errors.New("no project/config selected. Run `doppler setup`, pass --project and --config, or use a scoped service token"))
		}

		// Look up the requested engine in the registry. This indirection is the
		// pluggability seam: --engine selects which proxy implementation runs.
		factory, ok := proxy.Get(engineName)
		if !ok {
			utils.HandleError(fmt.Errorf("unknown proxy engine %q (available: %s)", engineName, strings.Join(proxy.Names(), ", ")))
		}

		// Resolve where the proxy keeps its data (CA) and writes its log. Create it
		// up front — on a fresh machine it doesn't exist yet, and the log file and
		// scaffolded config are written into it before the engine's own MkdirAll.
		dataDir := agentproxy.DefaultDataDir()
		if err := os.MkdirAll(dataDir, 0o700); err != nil {
			utils.HandleError(err, "unable to create the proxy data directory")
		}
		logPath, _ := cmd.Flags().GetString("log-file")
		if logPath == "" {
			logPath = filepath.Join(dataDir, "proxy.log")
		}
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			utils.HandleError(err, "unable to open proxy log file")
		}
		defer logFile.Close()

		// Build the engine, injecting the real Doppler-backed secret source.
		// Logs go to both the terminal and the log file.
		// Load the user-editable proxy config (scaffolding it, pre-filled with the
		// Anthropic passthrough, on first run). The --passthrough flag appends.
		proxyConfigPath, _ := cmd.Flags().GetString("proxy-config")
		if proxyConfigPath == "" {
			proxyConfigPath = filepath.Join(dataDir, "doppler-proxy.yaml")
		}
		proxyConfig, created, err := proxy.LoadOrScaffold(proxyConfigPath)
		if err != nil {
			utils.HandleError(err, "unable to load the proxy config")
		}
		if created {
			utils.Log(fmt.Sprintf("Created starter proxy config: %s", proxyConfigPath))
		} else {
			utils.Log(fmt.Sprintf("Proxy config: %s", proxyConfigPath))
		}
		utils.Log("  (edit it to set passthrough hosts, then restart)")

		// Address precedence: --address flag (if explicitly set) > config
		// listen_address > the flag's built-in default. The scaffolded default is
		// 0.0.0.0, which serves both host tools and the sandbox container; the
		// per-run proxy token (below) is what keeps a broad bind from being an open
		// proxy.
		if !cmd.Flags().Changed("address") && proxyConfig.ListenAddress != "" {
			address = proxyConfig.ListenAddress
		}

		flagPassthrough, _ := cmd.Flags().GetStringSlice("passthrough")
		passthrough := proxy.MergePassthrough(proxyConfig, flagPassthrough)
		upstreamProxy, _ := cmd.Flags().GetString("upstream-proxy")
		allowPrivateEgress, _ := cmd.Flags().GetBool("allow-private-egress")

		// Mint a per-run credential the proxy requires from every client, so a
		// broadly-bound or shared-network listener isn't an open forward proxy. It's
		// embedded in the agent env's proxy URL, so configured clients send it
		// automatically.
		proxyToken, err := mintProxyToken()
		if err != nil {
			utils.HandleError(err, "unable to generate the per-run proxy token")
		}

		opts, err := engineOptions(proxyConfig, proxyStartInputs{
			address:            address,
			dataDir:            dataDir,
			logOut:             io.MultiWriter(os.Stderr, logFile),
			passthrough:        passthrough,
			upstreamProxy:      upstreamProxy,
			proxyToken:         proxyToken,
			allowPrivateEgress: allowPrivateEgress,
			source:             proxy.NewDopplerSource(localConfig),
		})
		if err != nil {
			utils.HandleError(err, "invalid bindings in the proxy config")
		}
		warnShapeMismatches(opts.Binding, opts.Secrets)

		engine, err := factory(opts)
		if err != nil {
			utils.HandleError(err)
		}

		// Cancel the context on Ctrl-C / SIGTERM so the engine shuts down cleanly.
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		utils.Log(fmt.Sprintf("Starting proxy engine %q on %s (press Ctrl-C to stop)", engineName, address))
		utils.Log(fmt.Sprintf("Logs: %s", logPath))
		if err := engine.Start(ctx); err != nil {
			utils.HandleError(err)
		}
	},
}

// proxyStartInputs are the resolved flags proxy start turns into engine options.
type proxyStartInputs struct {
	address, dataDir, upstreamProxy, proxyToken string
	allowPrivateEgress                          bool
	passthrough                                 []string
	logOut                                      io.Writer
	source                                      agentproxy.SecretSource
}

// engineOptions is the one place config and flags become engine options, so a
// test can assert each setting actually reaches the engine.
func engineOptions(cfg *proxy.ProxyConfig, in proxyStartInputs) (proxy.Options, error) {
	binding, err := cfg.BindingResolver()
	if err != nil {
		return proxy.Options{}, err
	}
	secrets := agentproxy.NewRefreshingSource(in.source, agentproxy.RefreshOptions{
		Logf: func(format string, args ...any) { fmt.Fprintf(in.logOut, format+"\n", args...) },
	})
	return proxy.Options{
		ListenAddr:         in.address,
		Secrets:            secrets,
		DataDir:            in.dataDir,
		LogWriter:          in.logOut,
		AgentEnvPath:       agentproxy.AgentEnvPath(in.dataDir),
		PassthroughHosts:   in.passthrough,
		UpstreamProxy:      in.upstreamProxy,
		ProxyAuthToken:     in.proxyToken,
		Binding:            binding,
		AllowPrivateEgress: in.allowPrivateEgress,
	}, nil
}

// warnShapeMismatches logs each rule that points a recognizable token at another
// provider's host. The rule still wins at runtime, since a proxy or an enterprise
// host is a legitimate reason, but the mismatch is worth a look before the agent
// finds out.
func warnShapeMismatches(binding agentproxy.BindingResolver, secrets agentproxy.SecretSource) {
	rules, ok := binding.(*agentproxy.RuleResolver)
	if !ok {
		return
	}
	ctx := context.Background()
	names, err := secrets.List(ctx)
	if err != nil {
		return // the engine reports the load failure itself
	}
	values := make(map[string]string, len(names))
	for _, name := range names {
		if v, err := secrets.Fetch(ctx, agentproxy.SecretRef{Name: name}); err == nil {
			values[name] = v
		}
	}
	for _, warning := range rules.Validate(values) {
		utils.LogWarning(warning)
	}
}

func init() {
	proxyStartCmd.Flags().String("engine", "masked-hash", "proxy engine to run")
	proxyStartCmd.Flags().String("address", "0.0.0.0:14322", "address the proxy listens on; serves host + sandbox (set 127.0.0.1 for loopback-only, no sandbox). Overrides listen_address in the proxy config")
	proxyStartCmd.Flags().String("log-file", "", "write proxy logs to this file (default <data-dir>/proxy.log)")
	proxyStartCmd.Flags().String("proxy-config", "", "path to the proxy YAML config (default <data-dir>/doppler-proxy.yaml, scaffolded on first run)")
	proxyStartCmd.Flags().StringSlice("passthrough", nil, "extra hostnames to blind-tunnel, appended to the config's passthrough list")
	proxyStartCmd.Flags().String("upstream-proxy", "", "chain the proxy's own outbound connections through another HTTP proxy (e.g. http://127.0.0.1:3128 in a devcontainer)")
	proxyStartCmd.Flags().Bool("allow-private-egress", false, "let the proxy connect to loopback and private-network addresses (local development against a local upstream only)")
	// Project/config resolve from `doppler setup` scope by default; these flags
	// override it (same behavior as `doppler run`).
	proxyStartCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := proxyStartCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	proxyStartCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := proxyStartCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	proxyCmd.AddCommand(proxyStartCmd)
	rootCmd.AddCommand(proxyCmd)
}

// mintProxyToken returns a fresh, high-entropy per-run credential (256 bits, hex).
func mintProxyToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
