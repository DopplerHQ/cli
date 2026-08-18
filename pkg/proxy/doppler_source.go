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

	agentproxy "github.com/DopplerHQ/agent-proxy"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/models"
)

// dopplerSource is the real SecretSource: it reads the configured project/config
// from Doppler using the CLI's existing auth + API client — the same path
// `doppler run` uses. It fetches the config's secrets once (eagerly, on first
// use) and serves List/Fetch from that snapshot.
//
// This is the file the boundary promised would be the *only* change to make the
// proxy real — agent-proxy is untouched.
type dopplerSource struct {
	config models.ScopedOptions

	once    sync.Once
	secrets map[string]string
	loadErr error
}

// NewDopplerSource returns a SecretSource backed by the resolved CLI config.
func NewDopplerSource(config models.ScopedOptions) agentproxy.SecretSource {
	return &dopplerSource{config: config}
}

// load fetches the config's secrets exactly once.
func (s *dopplerSource) load() {
	s.once.Do(func() {
		computed, err := controllers.GetSecrets(s.config)
		if !err.IsNil() {
			s.loadErr = err.Unwrap()
			return
		}
		m := make(map[string]string, len(computed))
		for name, cs := range computed {
			if cs.ComputedValue != nil {
				m[name] = *cs.ComputedValue
			}
		}
		s.secrets = m
	})
}

func (s *dopplerSource) List(_ context.Context) ([]string, error) {
	s.load()
	if s.loadErr != nil {
		return nil, s.loadErr
	}
	names := make([]string, 0, len(s.secrets))
	for name := range s.secrets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func (s *dopplerSource) Fetch(_ context.Context, ref agentproxy.SecretRef) (string, error) {
	s.load()
	if s.loadErr != nil {
		return "", s.loadErr
	}
	value, ok := s.secrets[ref.Name]
	if !ok {
		return "", fmt.Errorf("secret %q not found in the configured Doppler config", ref.Name)
	}
	return value, nil
}
