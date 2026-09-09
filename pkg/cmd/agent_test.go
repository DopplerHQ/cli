/*
Copyright © 2026 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package cmd

import (
	"net"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"testing"
)

// The egress probes must cover BOTH IP families. A shared-box lock writes only
// iptables rules, so an IPv4-only probe list reports "contained" while the
// agent's IPv6 egress — external routes and `::1` services alike — is wide
// open. This asserts the wiring (both families, all IP literals) without dialing,
// so it can't go flaky; EgressBlockedTCP's own tests cover the dial behavior.
func TestEgressProbesCoverBothIPFamilies(t *testing.T) {
	var v4, v6External, v6Loopback bool
	for _, addr := range egressProbeTargets {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			t.Fatalf("probe %q is not a valid host:port: %v", addr, err)
		}
		ip := net.ParseIP(host)
		if ip == nil {
			t.Fatalf("probe %q must be an IP literal, got host %q (a hostname would fail at resolution and falsely look contained)", addr, host)
		}
		switch {
		case ip.To4() != nil:
			v4 = true
		case ip.IsLoopback():
			v6Loopback = true
		default:
			v6External = true
		}
	}
	if !v4 {
		t.Error("no IPv4 egress probe")
	}
	if !v6External {
		t.Error("no external IPv6 egress probe; an IPv4-only iptables lock leaves IPv6 egress open")
	}
	if !v6Loopback {
		t.Error("no IPv6 loopback egress probe; `::1` services bypass an IPv4-only lock")
	}
}

// TestProxyUserinfo: `agent enforce` must keep the per-run proxy token from
// agent.env's HTTPS_PROXY when it repoints the proxy host — dropping it 407s every
// agent request.
func TestProxyUserinfo(t *testing.T) {
	cases := []struct{ name, env, want string }{
		{"tokened", "HTTPS_PROXY='http://doppler:abc123@127.0.0.1:14322'\n", "doppler:abc123@"},
		{"tokened double-quoted", `HTTPS_PROXY="http://doppler:abc123@127.0.0.1:14322"`, "doppler:abc123@"},
		{"no userinfo", "HTTPS_PROXY='http://127.0.0.1:14322'\n", ""},
		{"no proxy line", "FOO=bar\nBAZ=qux\n", ""},
		{"http_proxy fallback", "HTTP_PROXY='http://doppler:xyz@127.0.0.1:14322'\n", "doppler:xyz@"},
	}
	for _, c := range cases {
		if got := proxyUserinfo([]byte(c.env)); got != c.want {
			t.Errorf("%s: proxyUserinfo = %q, want %q", c.name, got, c.want)
		}
	}
}

// The preflight opens the files that hold brokered secrets, by name. A directory
// would prove nothing, since one the agent cannot list still lets it open a file
// inside.
func TestCredentialSourcesAreConcreteFiles(t *testing.T) {
	got := credentialSources("/home/dev", "/home/dev/.config/agent-proxy")
	want := []string{
		filepath.Join("/home/dev", ".doppler", ".doppler.yaml"),
		filepath.Join("/home/dev", ".config", "agent-proxy", "ca.key"),
	}
	if !slices.Equal(got, want) {
		t.Fatalf("credentialSources = %v, want %v", got, want)
	}
	if len(credentialSources("", "")) != 0 {
		t.Fatal("with nothing resolved there is nothing to check")
	}
}

// Under sudo the developer is SUDO_USER, not the root that runs enforce.
func TestDeveloperHomeFollowsSudoUser(t *testing.T) {
	me, err := user.Current()
	if err != nil {
		t.Skip(err)
	}
	t.Setenv("SUDO_USER", me.Username)
	if got := developerHome(); got != me.HomeDir {
		t.Fatalf("developerHome under sudo = %q, want %q", got, me.HomeDir)
	}
	t.Setenv("SUDO_USER", "")
	home, _ := os.UserHomeDir()
	if got := developerHome(); got != home {
		t.Fatalf("developerHome without sudo = %q, want %q", got, home)
	}
}
