/*
Copyright © 2026 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package cmd

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	agentproxy "github.com/DopplerHQ/agent-proxy"
	"github.com/DopplerHQ/cli/pkg/proxy"
)

type staticSource map[string]string

func (s staticSource) List(context.Context) ([]string, error) {
	names := make([]string, 0, len(s))
	for n := range s {
		names = append(names, n)
	}
	return names, nil
}

func (s staticSource) Fetch(_ context.Context, ref agentproxy.SecretRef) (string, error) {
	return s[ref.Name], nil
}

// Every setting proxy start resolves has to reach the engine; a dropped field
// here would silently disable a feature.
func TestEngineOptionsCarryEverySetting(t *testing.T) {
	dir := t.TempDir()
	cfg := &proxy.ProxyConfig{Bindings: map[string][]agentproxy.Rule{"GH": {{Host: "api.github.com"}}}}
	opts, err := engineOptions(cfg, proxyStartInputs{
		address:            "127.0.0.1:14322",
		dataDir:            dir,
		logOut:             io.Discard,
		passthrough:        []string{"api.anthropic.com"},
		upstreamProxy:      "http://127.0.0.1:3128",
		proxyToken:         "per-run-token",
		allowPrivateEgress: true,
		source:             staticSource{"GH": "ghp_x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.AllowPrivateEgress || opts.ProxyAuthToken != "per-run-token" || opts.UpstreamProxy != "http://127.0.0.1:3128" || opts.ListenAddr != "127.0.0.1:14322" {
		t.Fatalf("flags did not reach the engine: %+v", opts)
	}
	if opts.AgentEnvPath != filepath.Join(dir, "agent.env") || opts.DataDir != dir {
		t.Fatalf("data dir paths wrong: %+v", opts)
	}
	if _, ok := opts.Secrets.(*agentproxy.RefreshingSource); !ok {
		t.Fatalf("secrets should be wrapped in RefreshingSource, got %T", opts.Secrets)
	}
	rules, ok := opts.Binding.(*agentproxy.RuleResolver)
	if !ok {
		t.Fatalf("binding should be the rule resolver, got %T", opts.Binding)
	}
	if allowed, _ := rules.Allowed(agentproxy.BindingRequest{Name: "GH", Value: "ghp_x", Dest: agentproxy.Destination{Host: "api.github.com:443", Path: "/", Method: "GET"}}); !allowed {
		t.Fatal("the declared rule should allow its host")
	}
	if allowed, _ := rules.Allowed(agentproxy.BindingRequest{Name: "DB", Value: "plain-value", Dest: agentproxy.Destination{Host: "db.example.com:443", Path: "/", Method: "GET"}}); allowed {
		t.Fatal("an undeclared secret must be refused by default")
	}
}

func TestEngineOptionsRejectUnknownUnboundPolicy(t *testing.T) {
	if _, err := engineOptions(&proxy.ProxyConfig{Unbound: "maybe"}, proxyStartInputs{logOut: io.Discard, source: staticSource{}}); err == nil {
		t.Fatal("an unknown unbound policy must be an error")
	}
}
