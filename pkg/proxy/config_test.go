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
	"strings"
	"testing"

	agentproxy "github.com/DopplerHQ/agent-proxy"
)

// On first run the scaffolded config pre-seeds the bindings section with the
// operator's own secret names (ENG-9770) — a commented stub per secret — so they
// edit real entries. The stubs stay commented, so a fresh scaffold injects nothing
// until a host is filled in.
func TestScaffoldSeedsBindingStubsFromSecretNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doppler-proxy.yaml")
	cfg, created, err := LoadOrScaffold(path, func() []string { return []string{"DATABASE_URL", "GITHUB_TOKEN"} })
	if err != nil || !created {
		t.Fatalf("scaffold: created=%v err=%v", created, err)
	}
	data, _ := os.ReadFile(path)
	for _, name := range []string{"DATABASE_URL", "GITHUB_TOKEN"} {
		if !strings.Contains(string(data), "#   "+name+":") {
			t.Errorf("scaffolded config missing a binding stub for %q\n%s", name, data)
		}
	}
	// The stubs are commented, so nothing is actually bound yet.
	if len(cfg.Bindings) != 0 {
		t.Errorf("scaffolded stubs must be commented (inactive), got bindings %v", cfg.Bindings)
	}
}

// With no secret names available, scaffolding falls back to the generic provider
// example rather than an empty bindings section.
func TestScaffoldFallsBackToExampleWithoutNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doppler-proxy.yaml")
	if _, _, err := LoadOrScaffold(path, nil); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "#   GITHUB_TOKEN:") {
		t.Errorf("fallback scaffold should carry the GITHUB_TOKEN example\n%s", data)
	}
}

func TestLoadOrScaffold(t *testing.T) {
	path := filepath.Join(t.TempDir(), "doppler-proxy.yaml")

	cfg, created, err := LoadOrScaffold(path, nil)
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
	cfg2, created2, err := LoadOrScaffold(path, nil)
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
	cfg, err := parseProxyConfig([]byte(buildStarterConfig(nil)))
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
	cfg, created, err := LoadOrScaffold(path, nil)
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

func TestParseMethods(t *testing.T) {
	cfg, err := parseProxyConfig([]byte(`
methods:
  OAUTH_SECRET:
    kind: oauth2_client_credentials
    token_url: https://provider.example.com/oauth/token
    client_id: cid
    scopes: [read, write]
  AWS_SECRET_ACCESS_KEY:
    kind: aws_sigv4
    service: s3
    region: us-west-2
    access_key_id: AWS_ACCESS_KEY_ID
`))
	if err != nil {
		t.Fatal(err)
	}
	o := cfg.Methods["OAUTH_SECRET"]
	if o.Kind != "oauth2_client_credentials" || o.TokenURL != "https://provider.example.com/oauth/token" || o.ClientID != "cid" || !slices.Equal(o.Scopes, []string{"read", "write"}) {
		t.Fatalf("oauth method = %+v", o)
	}
	a := cfg.Methods["AWS_SECRET_ACCESS_KEY"]
	if a.Kind != "aws_sigv4" || a.Service != "s3" || a.Region != "us-west-2" || a.AccessKeyID != "AWS_ACCESS_KEY_ID" {
		t.Fatalf("sigv4 method = %+v", a)
	}
	// snake_case yaml maps cleanly to the agent-proxy method registry.
	m := cfg.MethodConfigs()
	if m["OAUTH_SECRET"].TokenURL != "https://provider.example.com/oauth/token" || m["AWS_SECRET_ACCESS_KEY"].AccessKeyID != "AWS_ACCESS_KEY_ID" {
		t.Fatalf("MethodConfigs mapping wrong: %+v", m)
	}
}

func TestMethodConfigsNilWhenEmpty(t *testing.T) {
	if got := (&ProxyConfig{}).MethodConfigs(); got != nil {
		t.Fatalf("expected nil methods when none declared, got %v", got)
	}
}

func TestParsePassByValue(t *testing.T) {
	cfg, err := parseProxyConfig([]byte("pass_by_value:\n  - MODEL_TOKEN\n  - OTHER\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(cfg.PassByValue, []string{"MODEL_TOKEN", "OTHER"}) {
		t.Fatalf("pass_by_value = %v", cfg.PassByValue)
	}
}
