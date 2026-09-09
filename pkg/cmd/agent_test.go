/*
Copyright © 2026 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package cmd

import (
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"testing"
)

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
