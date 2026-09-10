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

// Package proxy is the CLI's integration layer for agent proxies. It defines
// the small Engine contract the CLI runs, a registry so `doppler proxy start
// --engine <name>` can pick an implementation, and the Doppler-backed
// capabilities (secret fetching, later auditing) injected into an engine.
//
// The proxy runtime itself lives in the separate github.com/DopplerTest/agent-proxy
// module; this package is where the CLI plugs into it.
package proxy

import (
	"context"
	"io"
	"sort"

	agentproxy "github.com/DopplerTest/agent-proxy"
)

// Engine is any runnable proxy implementation. The surface is intentionally
// tiny — just Start — so the CLI treats every engine interchangeably and can
// swap them via the --engine flag.
type Engine interface {
	Start(ctx context.Context) error
}

// Options is what the CLI hands to an engine factory: the capabilities it
// injects (today just the secret fetcher) plus operational settings. It grows
// as engines need more, without changing the Engine contract.
type Options struct {
	ListenAddr       string
	Secrets          agentproxy.SecretSource
	DataDir          string
	LogWriter        io.Writer
	AgentEnvPath     string
	PassthroughHosts []string
	UpstreamProxy    string
	// ProxyAuthToken is a per-run credential the CLI mints; the engine requires it
	// from every client (as a Basic Proxy-Authorization) and embeds it in the agent
	// env so standard clients send it automatically.
	ProxyAuthToken string
	// Binding authorizes each injection by destination. Nil means the engine's
	// own default.
	Binding agentproxy.BindingResolver
	// AllowPrivateEgress lets the proxy connect to loopback and private-network
	// addresses, for local development against a local upstream.
	AllowPrivateEgress bool
	// Methods declares a non-static credential brokering method per secret name
	// (OAuth2 client-credentials, AWS SigV4). Empty means every secret is static.
	Methods map[string]agentproxy.MethodConfig
	// PassByValue names the secrets written to the agent env as real values.
	PassByValue []string
}

// Factory builds an Engine from Options.
type Factory func(opts Options) (Engine, error)

// registry maps an engine name to its factory. Implementations populate it from
// their package init(), which is what makes engines pluggable.
//
// An Envoy engine was prototyped and is intentionally NOT shipped in this binary.
// It's preserved on the `austin/agent-proxy` branch (its adapter was pkg/proxy/
// envoy.go; the Envoy data plane lives in the agent-proxy repo's `envoy/` package on
// `austin/envoy-engine`). To bring it back, restore that adapter and its config
// surface — it self-registers here. See ai-proxy-docs/envoy-parked.md and ENG-9728.
var registry = map[string]Factory{}

// Register makes an engine available under name.
func Register(name string, f Factory) {
	registry[name] = f
}

// Get returns the factory registered under name.
func Get(name string) (Factory, bool) {
	f, ok := registry[name]
	return f, ok
}

// Names returns the registered engine names, sorted — handy for help text and
// "unknown engine" errors.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
