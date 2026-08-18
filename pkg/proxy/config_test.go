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

func TestMergePassthrough(t *testing.T) {
	cfg := &ProxyConfig{Passthrough: []string{"a.com", "b.com"}}
	got := MergePassthrough(cfg, []string{"b.com", "c.com", ""})
	want := []string{"a.com", "b.com", "c.com"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
