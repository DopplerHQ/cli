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
	"context"
	"fmt"
	"sort"
	"sync"

	agentproxy "github.com/DopplerTest/agent-proxy"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/models"
)

// dopplerSource is the real SecretSource: it reads the configured project/config
// from Doppler using the CLI's existing auth + API client — the same path
// `doppler run` uses. Every List fetches the config's secrets and serves the
// following Fetch calls from that snapshot, which is the shape RefreshingSource
// drives on each TTL.
type dopplerSource struct {
	config models.ScopedOptions

	mu      sync.Mutex
	secrets map[string]string
}

// NewDopplerSource returns a SecretSource backed by the resolved CLI config.
func NewDopplerSource(config models.ScopedOptions) agentproxy.SecretSource {
	return &dopplerSource{config: config}
}

// load fetches the config's secrets and replaces the snapshot.
func (s *dopplerSource) load() (map[string]string, error) {
	computed, err := controllers.GetSecrets(s.config)
	if !err.IsNil() {
		return nil, err.Unwrap()
	}
	m := make(map[string]string, len(computed))
	for name, cs := range computed {
		if cs.ComputedValue != nil {
			m[name] = *cs.ComputedValue
		}
	}
	s.mu.Lock()
	s.secrets = m
	s.mu.Unlock()
	return m, nil
}

func (s *dopplerSource) List(_ context.Context) ([]string, error) {
	m, err := s.load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *dopplerSource) Fetch(_ context.Context, ref agentproxy.SecretRef) (string, error) {
	s.mu.Lock()
	m := s.secrets
	s.mu.Unlock()
	if m == nil {
		var err error
		if m, err = s.load(); err != nil {
			return "", err
		}
	}
	value, ok := m[ref.Name]
	if !ok {
		return "", fmt.Errorf("secret %q not found in the configured Doppler config", ref.Name)
	}
	return value, nil
}
