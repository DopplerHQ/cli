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
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// ProxyConfig is the user-editable proxy configuration (doppler-proxy.yaml).
type ProxyConfig struct {
	// Passthrough lists hostnames the proxy blind-tunnels instead of
	// intercepting (no TLS termination, no injection).
	Passthrough []string `yaml:"passthrough"`
}

// starterConfig is written on first run so the operator has an editable file,
// pre-filled with sensible defaults (an AI agent's control-plane is passed
// through so its own traffic isn't intercepted).
const starterConfig = `# doppler-proxy.yaml — configuration for the Doppler agent credential proxy.
# Edit this file, then restart the proxy to apply changes.

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
`

// LoadOrScaffold loads the proxy config from path. If the file does not exist it
// writes the starter config and returns it with created=true.
func LoadOrScaffold(path string) (cfg *ProxyConfig, created bool, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(starterConfig), 0o644); err != nil {
			return nil, false, err
		}
		cfg, err = parseProxyConfig([]byte(starterConfig))
		return cfg, true, err
	}
	if err != nil {
		return nil, false, err
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
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, cfg.Passthrough...), extra...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
