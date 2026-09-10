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
	agentproxy "github.com/DopplerTest/agent-proxy"
)

// init registers the "masked-hash" engine: the per-secret-hash proxy backed by
// the agent-proxy runtime. The factory builds an agent-proxy Server, injecting
// the CLI's capabilities. Because *agentproxy.Server has a Start(ctx) method, it
// satisfies our Engine interface implicitly — no adapter needed.
//
// Additional engines register themselves the same way, which is what makes the
// --engine flag pluggable.
func init() {
	Register("masked-hash", func(opts Options) (Engine, error) {
		return agentproxy.New(agentproxy.Config{
			ListenAddr:         opts.ListenAddr,
			Secrets:            opts.Secrets,
			DataDir:            opts.DataDir,
			LogWriter:          opts.LogWriter,
			AgentEnvPath:       opts.AgentEnvPath,
			PassthroughHosts:   opts.PassthroughHosts,
			UpstreamProxy:      opts.UpstreamProxy,
			ProxyAuthToken:     opts.ProxyAuthToken,
			Binding:            opts.Binding,
			AllowPrivateEgress: opts.AllowPrivateEgress,
			Methods:            opts.Methods,
			PassByValue:        opts.PassByValue,
		})
	})
}
