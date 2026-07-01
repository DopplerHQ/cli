/*
Copyright © 2019 Doppler <support@doppler.com>

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
package http

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const testEtag = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"

func TestPollSecretsChangeCurrent(t *testing.T) {
	var capturedBodyLen int
	var capturedPath string
	var capturedUserAgent string
	var capturedMethod string
	var capturedAuth string
	var capturedIfNoneMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedUserAgent = r.Header.Get("user-agent")
		capturedMethod = r.Method
		capturedAuth = r.Header.Get("Authorization")
		capturedIfNoneMatch = r.Header.Get("If-None-Match")
		body, _ := io.ReadAll(r.Body)
		capturedBodyLen = len(body)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if !err.IsNil() {
		t.Fatalf("expected no error, got %+v", err)
	}
	if result != PollCurrent {
		t.Fatalf("expected PollCurrent, got %v", result)
	}
	if capturedPath != "/v4/secrets/poll" {
		t.Fatalf("expected path /v4/secrets/poll, got %s", capturedPath)
	}
	if capturedMethod != "GET" {
		t.Fatalf("expected GET method, got %s", capturedMethod)
	}
	if capturedAuth != "" {
		t.Fatalf("expected no Authorization header on poll requests, got %q", capturedAuth)
	}
	if capturedIfNoneMatch != `"`+testEtag+`"` {
		t.Fatalf("expected quoted etag in If-None-Match header, got %s", capturedIfNoneMatch)
	}
	if capturedBodyLen != 0 {
		t.Fatalf("expected empty request body, got %d bytes", capturedBodyLen)
	}
	if !strings.Contains(capturedUserAgent, "doppler-go-cli-") {
		t.Fatalf("expected user-agent header to contain 'doppler-go-cli-', got %s", capturedUserAgent)
	}
}

func TestPollSecretsChangeChanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if !err.IsNil() {
		t.Fatalf("expected no error, got %+v", err)
	}
	if result != PollChanged {
		t.Fatalf("expected PollChanged, got %v", result)
	}
}

func TestPollSecretsChangeBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.Code != http.StatusBadRequest {
		t.Fatalf("expected status code %d, got %d", http.StatusBadRequest, err.Code)
	}
}

func TestPollSecretsChangeIfNoneMatchQuoted(t *testing.T) {
	var capturedIfNoneMatch string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIfNoneMatch = r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	_, _ = PollSecretsChange(server.URL, true, testEtag)

	expected := `"` + testEtag + `"`
	if capturedIfNoneMatch != expected {
		t.Fatalf("expected If-None-Match header to be canonical quoted form %s, got %s", expected, capturedIfNoneMatch)
	}
	if !strings.HasPrefix(capturedIfNoneMatch, `"`) || !strings.HasSuffix(capturedIfNoneMatch, `"`) {
		t.Fatalf("expected If-None-Match header to start and end with a literal double-quote, got %s", capturedIfNoneMatch)
	}
}

func TestPollSecretsChangeServiceUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status code %d, got %d", http.StatusServiceUnavailable, err.Code)
	}
}

func TestPollSecretsChangeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.Code != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, err.Code)
	}
}

func TestPollSecretsChangeNotImplemented(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.Code != http.StatusNotImplemented {
		t.Fatalf("expected status code %d, got %d", http.StatusNotImplemented, err.Code)
	}
}

func TestPollSecretsChangeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// hang well beyond the expected ~2s client timeout
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	start := time.Now()
	result, err := PollSecretsChange(server.URL, true, testEtag)
	elapsed := time.Since(start)

	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.IsNil() {
		t.Fatalf("expected an error on timeout")
	}
	if elapsed < 1*time.Second || elapsed > 4*time.Second {
		t.Fatalf("expected request to time out around 2s, took %v", elapsed)
	}
}

func TestPollSecretsChangeTransportParity(t *testing.T) {
	// PollSecretsChange uses a dedicated short-timeout, no-retry client, but it must still
	// share the same proxy/DNS/TLS handling as the shared request() transport (built via
	// newTransport/newDialContext) rather than a bare, unconfigured http.Transport.
	req, err := http.NewRequest("GET", "https://example.com/v4/secrets/poll", nil)
	if err != nil {
		t.Fatalf("unable to build request: %v", err)
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	transport := newTransport(req, tlsConfig, newDialContext())

	if transport.Proxy == nil {
		t.Fatal("expected poll transport to have a non-nil Proxy func, matching shared request transport behavior")
	}
	if transport.DialContext == nil {
		t.Fatal("expected poll transport to have a non-nil DialContext func, matching shared request transport behavior")
	}
	if transport.TLSClientConfig != tlsConfig {
		t.Fatal("expected poll transport to use the provided TLS config")
	}
	if !transport.DisableKeepAlives {
		t.Fatal("expected poll transport to disable keep-alives, matching shared request transport behavior")
	}

	// with no HTTP(S)_PROXY env vars set, ProxyFromEnvironment resolves to no proxy for this
	// request; assert it agrees with the shared behavior rather than always erroring/differing
	proxyURL, proxyErr := transport.Proxy(req)
	sharedProxyURL, sharedErr := http.ProxyFromEnvironment(req)
	if proxyErr != nil || sharedErr != nil {
		t.Fatalf("unexpected error resolving proxy: poll=%v shared=%v", proxyErr, sharedErr)
	}
	if fmt.Sprint(proxyURL) != fmt.Sprint(sharedProxyURL) {
		t.Fatalf("expected poll transport proxy resolution to match shared http.ProxyFromEnvironment, got %v vs %v", proxyURL, sharedProxyURL)
	}
}

func TestPollSecretsChangeEtagNeverInURL(t *testing.T) {
	var capturedURL string
	var capturedRawQuery string
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		capturedRawQuery = r.URL.RawQuery
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	_, _ = PollSecretsChange(server.URL, true, testEtag)

	if strings.Contains(capturedURL, testEtag) {
		t.Fatalf("etag leaked into request URL: %s", capturedURL)
	}
	if capturedRawQuery != "" {
		t.Fatalf("expected no query params, got %s", capturedRawQuery)
	}

	// the etag is expected to appear in the quoted If-None-Match header (by design), but must
	// not leak into any other header (e.g. User-Agent or other client-* headers).
	for name, values := range capturedHeaders {
		if name == "If-None-Match" {
			continue
		}
		for _, value := range values {
			if strings.Contains(value, testEtag) {
				t.Fatalf("etag leaked into header %s: %s", name, value)
			}
		}
	}
}

func TestPollSecretsChangeGarbageStatusDoesNotEchoServerText(t *testing.T) {
	// A malicious/misbehaving poll endpoint that writes an unexpected status code alongside
	// response text (which is never read/parsed by the client). The resulting error must never
	// contain the etag or any server-written text -- only the numeric status code.
	serverText := "some-server-controlled-text-" + testEtag
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(599)
		_, _ = w.Write([]byte(serverText))
	}))
	defer server.Close()

	result, err := PollSecretsChange(server.URL, true, testEtag)
	if result != PollUnavailable {
		t.Fatalf("expected PollUnavailable, got %v", result)
	}
	if err.IsNil() {
		t.Fatalf("expected an error for garbage status code")
	}
	if err.Unwrap() != nil && strings.Contains(err.Unwrap().Error(), testEtag) {
		t.Fatalf("error must not contain the etag, got %q", err.Unwrap().Error())
	}
	if err.Unwrap() != nil && strings.Contains(err.Unwrap().Error(), serverText) {
		t.Fatalf("error must not contain server-written text, got %q", err.Unwrap().Error())
	}
	if strings.Contains(err.Message, testEtag) {
		t.Fatalf("error message must not contain the etag, got %q", err.Message)
	}
	if strings.Contains(err.Message, serverText) {
		t.Fatalf("error message must not contain server-written text, got %q", err.Message)
	}
}

func TestDownloadSecretsV4CapturesPollETag(t *testing.T) {
	var capturedPath string
	var capturedAuth string
	var capturedIfNoneMatch string
	var capturedQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedAuth = r.Header.Get("Authorization")
		capturedIfNoneMatch = r.Header.Get("If-None-Match")
		capturedQuery = r.URL.Query()
		w.Header().Set("X-Poll-ETag", "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"secrets":{}}`))
	}))
	defer server.Close()

	statusCode, headers, body, err := DownloadSecretsV4(server.URL, true, "apiKey123", "proj", "cfg", 0, nil, 0, nil)
	if !err.IsNil() {
		t.Fatalf("expected no error, got %+v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", statusCode)
	}
	if capturedPath != "/v4/configs/config/secrets/download" {
		t.Fatalf("expected path /v4/configs/config/secrets/download, got %s", capturedPath)
	}
	if capturedAuth != "Bearer apiKey123" {
		t.Fatalf("expected auth header, got %s", capturedAuth)
	}
	if capturedIfNoneMatch != "" {
		t.Fatalf("expected no If-None-Match header sent, got %s", capturedIfNoneMatch)
	}
	if got := capturedQuery.Get("project"); got != "proj" {
		t.Fatalf("expected project=proj, got %s", got)
	}
	if got := capturedQuery.Get("config"); got != "cfg" {
		t.Fatalf("expected config=cfg, got %s", got)
	}
	if got := capturedQuery.Get("include_dynamic_secrets"); got != "true" {
		t.Fatalf("expected include_dynamic_secrets=true, got %s", got)
	}
	pollETag := headers.Get("X-Poll-ETag")
	if pollETag != "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234" {
		t.Fatalf("expected X-Poll-ETag header to be captured, got %s", pollETag)
	}
	if !strings.Contains(string(body), "secrets") {
		t.Fatalf("expected response body to be returned, got %s", string(body))
	}
}

func TestDownloadSecretsV4NoPollETagWhenAbsent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"secrets":{}}`))
	}))
	defer server.Close()

	_, headers, _, err := DownloadSecretsV4(server.URL, true, "apiKey123", "proj", "cfg", 0, nil, 0, nil)
	if !err.IsNil() {
		t.Fatalf("expected no error, got %+v", err)
	}
	if headers.Get("X-Poll-ETag") != "" {
		t.Fatalf("expected empty X-Poll-ETag header, got %s", headers.Get("X-Poll-ETag"))
	}
}

func TestDownloadSecretsV4FallbackStatusCodes(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusNotImplemented} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))

		statusCode, _, _, _ := DownloadSecretsV4(server.URL, true, "apiKey123", "proj", "cfg", 0, nil, 0, nil)
		if statusCode != code {
			t.Fatalf("expected status code %d, got %d", code, statusCode)
		}
		server.Close()
	}
}

func TestDownloadSecretsV4DynamicSecretsTTLAndSecretsParams(t *testing.T) {
	var capturedQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	_, _, _, err := DownloadSecretsV4(server.URL, true, "apiKey123", "proj", "cfg", 0, nil, 30*time.Second, []string{"FOO", "BAR"})
	if !err.IsNil() {
		t.Fatalf("expected no error, got %+v", err)
	}
	if got := capturedQuery.Get("dynamic_secrets_ttl_sec"); got != "30" {
		t.Fatalf("expected dynamic_secrets_ttl_sec=30, got %s", got)
	}
	if got := capturedQuery.Get("secrets"); got != "FOO,BAR" {
		t.Fatalf("expected secrets=FOO,BAR, got %s", got)
	}
}
