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
	"os"
	"path/filepath"
	"slices"
	"testing"

	agentproxy "github.com/DopplerHQ/agent-proxy"
)

func TestLoadOrScaffold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doppler-proxy.yaml")

	cfg, created, err := LoadOrScaffold(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("expected the config to be scaffolded on first run")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file was not written: %v", err)
	}
	if !slices.Contains(cfg.Passthrough, "api.anthropic.com") {
		t.Fatalf("starter config missing api.anthropic.com; got %v", cfg.Passthrough)
	}
	if cfg.ListenAddress != "0.0.0.0:14322" {
		t.Fatalf("starter config listen_address = %q, want 0.0.0.0:14322", cfg.ListenAddress)
	}

	// A second load reads the existing file — not scaffolded again.
	cfg2, created2, err := LoadOrScaffold(path)
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Fatal("expected created=false when the file already exists")
	}
	if !slices.Equal(cfg.Passthrough, cfg2.Passthrough) {
		t.Fatal("passthrough changed across reloads")
	}
}

// ENG-9723: the scaffolded passthrough list is a set of blind holes — no audit, no
// injection — so it must stay minimal. A third-party error sink (sentry.io) or
// telemetry (statsig.anthropic.com) must not be blind-tunneled: they work fine
// intercepted, and a Sentry DSN is a world-writable exfil endpoint.
func TestScaffoldedPassthroughDropsTelemetryHoles(t *testing.T) {
	cfg, err := parseProxyConfig([]byte(starterConfig))
	if err != nil {
		t.Fatal(err)
	}
	banned := map[string]string{
		"sentry.io":             "a third-party, world-writable error sink",
		"statsig.anthropic.com": "telemetry",
	}
	for _, h := range cfg.Passthrough {
		if why, bad := banned[h]; bad {
			t.Errorf("passthrough must not blind-tunnel %q (%s) — it works intercepted", h, why)
		}
	}
	if !slices.Contains(cfg.Passthrough, "api.anthropic.com") {
		t.Error("api.anthropic.com must remain — the agent cannot function without its model API")
	}
}

func TestLoadOrScaffoldRewritesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doppler-proxy.yaml")
	// Pre-create an empty (blank) file — the bug case.
	if err := os.WriteFile(path, []byte("   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, created, err := LoadOrScaffold(path)
	if err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("an empty file should be (re)scaffolded, created=true")
	}
	if !slices.Contains(cfg.Passthrough, "api.anthropic.com") {
		t.Fatalf("scaffolded config not populated; got %v", cfg.Passthrough)
	}
	data, _ := os.ReadFile(path)
	if len(data) == 0 {
		t.Fatal("file is still empty after scaffold")
	}
}

func TestMergePassthrough(t *testing.T) {
	cfg := &ProxyConfig{Passthrough: []string{"a.com", "b.com"}}
	got := MergePassthrough(cfg, []string{"b.com", "c.com", ""})
	want := []string{"a.com", "b.com", "c.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParsePassthroughList(t *testing.T) {
	cfg, err := parseProxyConfig([]byte("passthrough:\n  - a.com\n  - b.com\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.Passthrough, []string{"a.com", "b.com"}) {
		t.Fatalf("passthrough = %v", cfg.Passthrough)
	}
}

func TestParseBindings(t *testing.T) {
	cfg, err := parseProxyConfig([]byte(`
bindings:
  GITHUB_TOKEN:
    - host: api.github.com
      paths: ["/repos/**"]
      methods: [GET]
  STRIPE_KEY:
    - host: api.stripe.com
unbound: trust-first-use
`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Bindings) != 2 {
		t.Fatalf("bindings = %v", cfg.Bindings)
	}
	gh := cfg.Bindings["GITHUB_TOKEN"]
	if len(gh) != 1 || gh[0].Host != "api.github.com" || !slices.Equal(gh[0].Paths, []string{"/repos/**"}) || !slices.Equal(gh[0].Methods, []string{"GET"}) {
		t.Fatalf("GITHUB_TOKEN rules = %+v", gh)
	}
	if cfg.Unbound != "trust-first-use" {
		t.Fatalf("unbound = %q", cfg.Unbound)
	}
	if _, err := cfg.BindingResolver(); err != nil {
		t.Fatal(err)
	}
}

// With no bindings block at all, an unrecognizable secret is refused everywhere.
func TestBindingResolverDefaultsToDeny(t *testing.T) {
	r, err := (&ProxyConfig{}).BindingResolver()
	if err != nil {
		t.Fatal(err)
	}
	ok, why := r.Allowed(agentproxy.BindingRequest{
		Name:  "DB_PASSWORD",
		Value: "plain-database-password",
		Dest:  agentproxy.Destination{Host: "db.example.com:443", Path: "/", Method: "GET"},
	})
	if ok {
		t.Fatal("an undeclared secret must be refused by default")
	}
	if why == "" {
		t.Fatal("refusal should carry a reason")
	}
}

func TestBindingResolverRejectsUnknownPolicy(t *testing.T) {
	if _, err := (&ProxyConfig{Unbound: "maybe"}).BindingResolver(); err == nil {
		t.Fatal("an unknown unbound policy must be an error")
	}
}
