/*
Copyright © 2026 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
*/

package cmd

import "testing"

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
