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
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	agentproxy "github.com/DopplerHQ/agent-proxy"
	"github.com/DopplerHQ/agent-proxy/enforce"
	"github.com/DopplerHQ/agent-proxy/sandbox"
	"github.com/DopplerHQ/agent-proxy/verify"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run AI agents against the credential proxy (experimental)",
	Args:  cobra.NoArgs,
}

var agentRunCmd = &cobra.Command{
	Use:   "run -- <command>",
	Short: "Run a command inside a locked-down sandbox whose only egress is the proxy",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		proxyPort, _ := cmd.Flags().GetInt("proxy-port")
		rebuild, _ := cmd.Flags().GetBool("rebuild")
		dockerBin, _ := cmd.Flags().GetString("docker")

		// Resolve the proxy's artifacts using the shared path helpers.
		dataDir := agentproxy.DefaultDataDir()
		caPath := agentproxy.CACertPath(dataDir)
		envPath := agentproxy.AgentEnvPath(dataDir)

		for _, p := range []string{caPath, envPath} {
			if _, err := os.Stat(p); err != nil {
				utils.HandleError(fmt.Errorf(
					"proxy artifacts not found (%s). Start the proxy first, bound to an address the sandbox can reach:\n  doppler proxy start --address 0.0.0.0:%d",
					p, proxyPort))
			}
		}

		cfg := sandbox.Config{
			ProxyPort:    proxyPort,
			CACertPath:   caPath,
			AgentEnvPath: envPath,
			Command:      args,
			DockerBin:    dockerBin,
			Interactive:  true,
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		if rebuild {
			utils.Log("Rebuilding sandbox image…")
			if err := sandbox.BuildImage(ctx, cfg); err != nil {
				utils.HandleError(err, "failed to build the sandbox image")
			}
		} else {
			utils.Log("Preparing sandbox image (first run may take a few minutes)…")
			if err := sandbox.EnsureImage(ctx, cfg); err != nil {
				utils.HandleError(err, "failed to prepare the sandbox image")
			}
		}

		if err := sandbox.Run(ctx, cfg); err != nil {
			utils.HandleError(err, "sandbox exited with an error")
		}
	},
}

// agentDoctorCmd verifies the sandbox contract for the environment it's run in.
// It is the same verifier the enforced paths invoke internally as a preflight;
// as a standalone command it doubles as a diagnostic ("why can't the agent reach
// GitHub?"). Run it AS the agent — same user, network, and env the agent gets.
var agentDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Verify the sandbox contract (egress containment, CA trust, privilege, hygiene)",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		enforced, _ := cmd.Flags().GetBool("enforced")
		strictDNS, _ := cmd.Flags().GetBool("strict-dns")
		testURL, _ := cmd.Flags().GetString("test-url")

		// The proxy the agent is meant to use: its HTTPS_PROXY, falling back to
		// the default listen address.
		proxyURL, _ := cmd.Flags().GetString("proxy")
		if proxyURL == "" {
			if v := firstEnv("HTTPS_PROXY", "https_proxy"); v != "" {
				proxyURL = v
			} else {
				proxyURL = "http://127.0.0.1:14322"
			}
		}

		// The proxy CA: prefer an explicit flag, then the vars the agent trusts,
		// then the default on-disk location.
		caPath, _ := cmd.Flags().GetString("ca")
		if caPath == "" {
			if v := firstEnv("NODE_EXTRA_CA_CERTS", "CURL_CA_BUNDLE", "SSL_CERT_FILE"); v != "" {
				caPath = v
			} else {
				caPath = agentproxy.CACertPath(agentproxy.DefaultDataDir())
			}
		}

		report := verify.Doctor{Enforced: enforced, Checks: agentChecks(proxyURL, caPath, strictDNS, testURL, resolveCredentialSources(cmd))}.Run()
		report.Render(os.Stdout)
		os.Exit(report.ExitCode())
	},
}

// agentChecks is the standard contract check-list, shared by `agent doctor` and
// the preflight `agent enforce` runs before launching the agent — so both assert
// exactly the same contract.
// egressProbeTargets are the IP:port literals clause 1 proves are directly
// unreachable from the agent. They span both IP families on purpose: a
// shared-box lock that only writes iptables rules leaves the agent's IPv6
// egress wide open wherever the container has an IPv6 route, so an IPv4-only
// probe list reports "contained" on a box that isn't. The list mixes external
// routes (the agent must not reach the internet directly) with an IPv6 loopback
// service port (a `::1` Postgres or the like is egress the lock must also cut,
// and netfilter's IPv4 chain never sees it). Every entry is an IP literal, never
// a hostname — a blocked resolver would make a hostname dial fail at resolution
// and falsely look contained.
var egressProbeTargets = []string{
	"1.1.1.1:443",                // IPv4 external
	"8.8.8.8:443",                // IPv4 external
	"1.1.1.1:80",                 // IPv4 external (plaintext)
	"[2606:4700:4700::1111]:443", // IPv6 external — an IPv4-only iptables lock never covers this
	"[::1]:5432",                 // IPv6 loopback — a local service (e.g. Postgres) the agent must not reach
}

func agentChecks(proxyURL, caPath string, strictDNS bool, testURL string, credentialSources []string) []verify.Check {
	// clause 1 — egress containment (adversarial: dial by IP literal, both families)
	var checks []verify.Check
	for _, addr := range egressProbeTargets {
		checks = append(checks, verify.EgressBlockedTCP(addr))
	}
	checks = append(checks,
		verify.EgressDNS("8.8.8.8:53", strictDNS),
		// proxy reachability
		verify.ProxyReachable(proxyURL),
		// clause 3 — CA trust
		verify.CACertValid(caPath),
		verify.CATrustEnv(),
		verify.CAEndToEnd(proxyURL, testURL),
	)
	// clause 2 — privilege
	checks = append(checks, privilegeChecks()...)
	// credential hygiene (Doppler-specific)
	checks = append(checks,
		verify.EnvAbsent("DOPPLER_TOKEN"),
		verify.EnvNoTokenShapes("real token shapes", "dp.st.", "dp.pt."),
	)
	// masking only holds while the agent cannot read the brokered secrets off disk
	for _, p := range credentialSources {
		checks = append(checks, verify.FileUnreadable("agent cannot read "+p, p))
	}
	return checks
}

// privilegeChecks proves clause 2: the agent runs unprivileged AND cannot regain
// the capability it would need to unlock its own egress. A clean effective set
// (NetAdminAbsent) is not enough on its own — while CAP_NET_ADMIN remains in the
// bounding set, a file-capability or setuid binary can hand it back — so the
// bounding set must be clean too (NetAdminNotAcquirable, ENG-9749).
func privilegeChecks() []verify.Check {
	return []verify.Check{
		verify.UIDNotRoot(),
		verify.NetAdminAbsent(),
		verify.NetAdminNotAcquirable(),
	}
}

// developerHome is the home of the person whose secrets the proxy brokers: the
// user behind sudo under `agent enforce`, otherwise the current user.
func developerHome() string {
	if dev := os.Getenv("SUDO_USER"); dev != "" {
		if u, err := user.Lookup(dev); err == nil {
			return u.HomeDir
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return ""
}

// credentialSources are the files holding what the proxy brokers on the agent's
// behalf: the developer's Doppler config and the proxy CA key. Files rather
// than their directories, since a directory the agent cannot list still lets it
// open a file inside by name.
func credentialSources(devHome, dataDir string) []string {
	var out []string
	if devHome != "" {
		out = append(out, filepath.Join(devHome, ".doppler", ".doppler.yaml"))
	}
	if dataDir != "" {
		out = append(out, filepath.Join(dataDir, "ca.key"))
	}
	return out
}

// dataDirUnder is agentproxy.DefaultDataDir for another user's home, following
// the platform default. --proxy-data-dir covers an XDG_CONFIG_HOME override.
func dataDirUnder(home string) string {
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "agent-proxy")
	}
	return filepath.Join(home, ".config", "agent-proxy")
}

// agentEnforceCmd installs the sandbox contract IN PLACE — inside a box the user
// already has (a devcontainer, a VM) — then runs the agent. It locks the agent's
// egress to only the proxy, drops to an unprivileged user, runs the doctor
// preflight, and execs the command. Must be run as root (e.g. via sudo, or from
// a devcontainer feature's init). Linux only.
var agentEnforceCmd = &cobra.Command{
	Use:   "enforce -- <command>",
	Short: "Lock egress to the proxy in place, drop privileges, and run the agent (Linux, root)",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		strategyName, _ := cmd.Flags().GetString("strategy")
		agentUser, _ := cmd.Flags().GetString("agent-user")
		proxyHost, _ := cmd.Flags().GetString("proxy-host")
		proxyPort, _ := cmd.Flags().GetInt("proxy-port")
		strictDNS, _ := cmd.Flags().GetBool("strict-dns")
		testURL, _ := cmd.Flags().GetString("test-url")

		var strat enforce.Strategy
		switch strategyName {
		case "owned-container":
			strat = enforce.OwnedContainer{}
		case "shared-box", "":
			strat = enforce.SharedBox{}
		default:
			utils.HandleError(fmt.Errorf("unknown strategy %q (want owned-container or shared-box)", strategyName))
		}

		// Resolve the unprivileged agent user we'll drop to.
		u, err := user.Lookup(agentUser)
		if err != nil {
			utils.HandleError(fmt.Errorf("agent user %q not found: %w. Create it (the devcontainer feature does this) or pass --agent-user", agentUser, err))
		}
		uid, gid, groups := resolveUser(u)

		// The firewall rule needs an IP; the proxy env keeps the host name.
		proxyIP := proxyHost
		if net.ParseIP(proxyHost) == nil {
			ips, err := net.LookupHost(proxyHost)
			if err != nil || len(ips) == 0 {
				utils.HandleError(fmt.Errorf("could not resolve proxy host %q: %w", proxyHost, err))
			}
			proxyIP = ips[0]
		}

		// CA path: flag, else default on-disk location.
		caPath, _ := cmd.Flags().GetString("ca")
		if caPath == "" {
			caPath = agentproxy.CACertPath(agentproxy.DefaultDataDir())
		}

		// Build the agent env from the proxy's agent.env, repointing the proxy and
		// CA vars at this boundary and stripping anything the agent must not hold.
		envPath, _ := cmd.Flags().GetString("agent-env")
		if envPath == "" {
			envPath = agentproxy.AgentEnvPath(agentproxy.DefaultDataDir())
		}
		rawEnv, err := os.ReadFile(envPath)
		if err != nil {
			utils.HandleError(fmt.Errorf("reading agent env %s: %w. Start the proxy first", envPath, err))
		}
		// Keep the per-run proxy token (userinfo) from agent.env's HTTPS_PROXY and
		// repoint only the host at this boundary. Dropping it would hand the agent a
		// credential-less proxy URL and every request would get a 407.
		proxyURL := fmt.Sprintf("http://%s%s:%d", proxyUserinfo(rawEnv), proxyHost, proxyPort)
		overrides := map[string]string{
			"HTTPS_PROXY":         proxyURL,
			"HTTP_PROXY":          proxyURL,
			"NODE_EXTRA_CA_CERTS": caPath,
			"CURL_CA_BUNDLE":      caPath,
			"SSL_CERT_FILE":       caPath,
			// git and Python's requests honor their own CA vars, not the three above;
			// without these, git over HTTPS to an intercepted host fails in the enforced
			// box now that enforce no longer installs the CA into the system trust store.
			"GIT_SSL_CAINFO":     caPath,
			"REQUESTS_CA_BUNDLE": caPath,
			// Enforce clears the environment before exec, so the essential process
			// vars for the dropped-privilege agent must be set explicitly.
			"HOME":    u.HomeDir,
			"USER":    agentUser,
			"LOGNAME": agentUser,
			"PATH":    envOr("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"),
			"TERM":    envOr("TERM", "xterm"),
		}
		env := enforce.ParseAgentEnv(string(rawEnv))
		env = enforce.OverrideEnv(env, overrides)
		env = enforce.RemoveEnv(env, "DOPPLER_TOKEN", "NO_PROXY", "no_proxy")

		// Resolved here, while SUDO_USER is still in the environment; Enforce clears
		// the environment before the preflight runs as the agent.
		sources := resolveCredentialSources(cmd)

		// The preflight is the same contract doctor asserts, run as the agent user
		// after the lock. It fails the launch if the sandbox isn't sound.
		preflight := func() error {
			rep := verify.Doctor{Enforced: true, Checks: agentChecks(proxyURL, caPath, strictDNS, testURL, sources)}.Run()
			rep.Render(os.Stderr)
			if rep.Failed() {
				return errors.New("sandbox contract check failed; refusing to launch the agent")
			}
			return nil
		}

		// No CA path is passed: Enforce no longer installs a system-trust CA (ENG-9745);
		// the agent env's CA vars carry that trust instead.
		err = enforce.Enforce(enforce.Config{
			Strategy:    strat,
			Params:      enforce.Params{ProxyIP: proxyIP, ProxyPort: proxyPort, AgentUID: uid},
			AgentUID:    uid,
			AgentGID:    gid,
			AgentGroups: groups,
			Env:         env,
			Command:     args,
			Preflight:   preflight,
			Logf:        func(f string, a ...any) { utils.Log(fmt.Sprintf(f, a...)) },
		})
		if err != nil {
			utils.HandleError(err, "enforce failed")
		}
	},
}

// resolveCredentialSources reads the developer's home and the proxy data dir
// from the current environment and flags.
func resolveCredentialSources(cmd *cobra.Command) []string {
	devHome := developerHome()
	dataDir, _ := cmd.Flags().GetString("proxy-data-dir")
	if dataDir == "" {
		dataDir = dataDirUnder(devHome)
	}
	return credentialSources(devHome, dataDir)
}

// resolveUser turns an os/user.User into numeric uid/gid and supplementary gids.
func resolveUser(u *user.User) (uid, gid int, groups []int) {
	uid, _ = strconv.Atoi(u.Uid)
	gid, _ = strconv.Atoi(u.Gid)
	if gidStrs, err := u.GroupIds(); err == nil {
		for _, g := range gidStrs {
			if n, err := strconv.Atoi(g); err == nil {
				groups = append(groups, n)
			}
		}
	}
	if len(groups) == 0 {
		groups = []int{gid}
	}
	return uid, gid, groups
}

// firstEnv returns the first non-empty value among the given env var names.
func firstEnv(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// envOr returns the env var's value, or fallback if it's unset/empty.
func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

// proxyUserinfo returns the "user:pass@" prefix from the agent env's HTTPS_PROXY
// (the per-run proxy token), or "" if none. Used so `agent enforce` keeps the
// credential when it repoints the proxy host, instead of dropping it.
func proxyUserinfo(rawEnv []byte) string {
	for _, line := range strings.Split(string(rawEnv), "\n") {
		line = strings.TrimSpace(line)
		v, ok := strings.CutPrefix(line, "HTTPS_PROXY=")
		if !ok {
			v, ok = strings.CutPrefix(line, "HTTP_PROXY=")
		}
		if !ok {
			continue
		}
		v = strings.Trim(v, `'"`) // agent.env shell-quotes values
		if u, err := url.Parse(v); err == nil && u.User != nil {
			return u.User.String() + "@"
		}
	}
	return ""
}

func init() {
	agentRunCmd.Flags().Int("proxy-port", 14322, "port the credential proxy is listening on")
	agentRunCmd.Flags().Bool("rebuild", false, "rebuild the sandbox image before running")
	agentRunCmd.Flags().String("docker", "docker", "container CLI to use (docker, podman, ...)")
	agentCmd.AddCommand(agentRunCmd)

	agentDoctorCmd.Flags().Bool("enforced", false, "assert the full contract: an egress-containment failure is fatal")
	agentDoctorCmd.Flags().Bool("strict-dns", false, "treat an open external DNS resolver as a failure, not a warning")
	agentDoctorCmd.Flags().String("proxy", "", "proxy URL the agent should use (default $HTTPS_PROXY or http://127.0.0.1:14322)")
	agentDoctorCmd.Flags().String("ca", "", "proxy CA cert path (default $NODE_EXTRA_CA_CERTS or <data-dir>/ca.crt)")
	agentDoctorCmd.Flags().String("test-url", "https://example.com", "URL fetched through the proxy to test end-to-end CA trust")
	agentDoctorCmd.Flags().String("proxy-data-dir", "", "proxy data directory holding the CA key (default: the developer's platform config dir)")
	agentCmd.AddCommand(agentDoctorCmd)

	agentEnforceCmd.Flags().String("strategy", "shared-box", "egress lock strategy: shared-box (compose onto an existing firewall) or owned-container (flush)")
	agentEnforceCmd.Flags().String("agent-user", "agent", "unprivileged user to drop to before running the agent")
	agentEnforceCmd.Flags().String("proxy-host", "127.0.0.1", "host the credential proxy is reachable at from inside this boundary")
	agentEnforceCmd.Flags().Int("proxy-port", 14322, "port the credential proxy is listening on")
	agentEnforceCmd.Flags().String("ca", "", "proxy CA cert path (default <data-dir>/ca.crt)")
	agentEnforceCmd.Flags().String("agent-env", "", "path to the proxy's agent.env (default <data-dir>/agent.env)")
	agentEnforceCmd.Flags().Bool("strict-dns", false, "treat an open external DNS resolver as a preflight failure")
	agentEnforceCmd.Flags().String("test-url", "https://example.com", "URL fetched through the proxy to test end-to-end CA trust")
	agentEnforceCmd.Flags().String("proxy-data-dir", "", "proxy data directory holding the CA key (default: the developer's platform config dir)")
	agentCmd.AddCommand(agentEnforceCmd)

	rootCmd.AddCommand(agentCmd)
}
