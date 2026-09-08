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

package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	agentproxy "github.com/DopplerHQ/agent-proxy"
	"gopkg.in/yaml.v3"
)

// ProxyConfig is the user-editable proxy configuration (doppler-proxy.yaml).
type ProxyConfig struct {
	// ListenAddress is the address the proxy binds. Defaults (via the starter
	// config) to 0.0.0.0:14322 so the `doppler agent run` sandbox can reach it.
	// The --address flag overrides this.
	ListenAddress string `yaml:"listen_address"`

	// Passthrough lists hostnames the proxy blind-tunnels instead of
	// intercepting (no TLS termination, no injection).
	Passthrough []string `yaml:"passthrough"`

	// Bindings declares where each secret may be injected, by secret name. A
	// secret with no entry falls under Unbound.
	Bindings map[string][]agentproxy.Rule `yaml:"bindings"`

	// Unbound is the policy for a secret with no bindings entry: "deny" (the
	// default) refuses it everywhere, "trust-first-use" pins it to the first
	// host the agent sends it to.
	Unbound string `yaml:"unbound"`
}

// BindingResolver builds the resolver the proxy authorizes injection with.
func (c *ProxyConfig) BindingResolver() (agentproxy.BindingResolver, error) {
	var policy agentproxy.UnboundPolicy
	switch c.Unbound {
	case "", "deny":
		policy = agentproxy.UnboundDeny
	case "trust-first-use":
		policy = agentproxy.UnboundTOFU
	default:
		return nil, fmt.Errorf("unbound must be deny or trust-first-use, got %q", c.Unbound)
	}
	return agentproxy.NewRuleResolver(c.Bindings, policy), nil
}

// starterConfig is written on first run so the operator has an editable file,
// pre-filled with sensible defaults (an AI agent's control-plane is passed
// through so its own traffic isn't intercepted).
const starterConfig = `# doppler-proxy.yaml — configuration for the Doppler agent credential proxy.
# Edit this file, then restart the proxy to apply changes.

# Address the proxy listens on. 0.0.0.0 serves both host tools (via 127.0.0.1) and
# the ` + "`doppler agent run`" + ` sandbox container (via the docker bridge). Every client
# must present the per-run proxy token, so a broad bind is not an open proxy. Set
# 127.0.0.1 to bind loopback only (the sandbox container cannot reach that).
# --address overrides this.
listen_address: 0.0.0.0:14322

# Hosts the proxy BLIND-TUNNELS instead of intercepting: no TLS termination and
# no credential injection. Put an agent's own control-plane here so its traffic
# passes through untouched (e.g. an AI agent reaching its model provider). This
# must include the agent's AUTH domains too — intercepting them breaks login
# (auth endpoints reject an unexpected CA), so Claude's login/session domains are
# passed through alongside its model endpoint.
passthrough:
  - api.anthropic.com
  - console.anthropic.com
  - claude.ai
  - claude.com
  - statsig.anthropic.com
  - sentry.io

# Where each secret may be injected. A secret with no entry is refused everywhere
# unless unbound below says otherwise. paths are globs matched per segment (** spans
# segments) and methods are optional; both default to any.
# bindings:
#   GITHUB_TOKEN:
#     - host: api.github.com
#       paths: ["/repos/**", "/user"]
#       methods: [GET, POST]
#   STRIPE_KEY:
#     - host: api.stripe.com

# Policy for a secret with no bindings entry. deny refuses it everywhere and
# logs the host it was sent to. trust-first-use pins it to the first host the
# agent uses, which lets the agent decide where the credential goes.
# unbound: deny
`

// LoadOrScaffold loads the proxy config from path. If the file does not exist it
// writes the starter config and returns it with created=true.
func LoadOrScaffold(path string) (cfg *ProxyConfig, created bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, false, err
	}
	// Write the starter config when the file is missing OR empty — so a stray blank
	// file (e.g. from an interrupted write) still gets populated on startup instead
	// of silently loading as an empty config.
	if errors.Is(err, os.ErrNotExist) || len(bytes.TrimSpace(data)) == 0 {
		if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
			return nil, false, err
		}
		cfg, err = parseProxyConfig([]byte(starterConfig))
		return cfg, true, err
	}
	cfg, err = parseProxyConfig(data)
	return cfg, false, err
}

func parseProxyConfig(data []byte) (*ProxyConfig, error) {
	var cfg ProxyConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// MergePassthrough returns the config's passthrough hosts plus any extras,
// de-duplicated and order-preserving (config entries first).
func MergePassthrough(cfg *ProxyConfig, extra []string) []string {
	return mergeHostLists(cfg.Passthrough, extra)
}

func mergeHostLists(base, extra []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, base...), extra...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
