/*
Copyright © 2024 Doppler <support@doppler.com>

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
package controllers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/stretchr/testify/assert"
)

const (
	fetchTestPassphrase = "test-passphrase"
	fetchTestProject    = "proj"
	fetchTestConfig     = "cfg"
	fetchTestPollETag   = "poll-etag-abc123"
)

// endpointCounts tracks how many times each API path was hit within a single test server.
type endpointCounts struct {
	poll         int
	v4Download   int
	v3Download   int
	lastV4ETag   string
	v3RespETag   string // if non-empty, v3 responds with this ETag header
	v4StatusCode int    // status the v4 download endpoint returns (default 200)
	pollStatus   int    // HTTP status the poll endpoint returns (default 304, meaning "current")
}

// newFetchTestServer builds an httptest server routing the three secrets endpoints and
// recording hit counts. v3 always returns 200 with the given body so the v3 fallback path
// never triggers utils.HandleError (which calls os.Exit).
func newFetchTestServer(t *testing.T, counts *endpointCounts, v3Body []byte, v4Body []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v4/secrets/poll":
			counts.poll++
			status := counts.pollStatus
			if status == 0 {
				status = http.StatusNotModified
			}
			w.WriteHeader(status)
		case "/v4/configs/config/secrets/download":
			counts.v4Download++
			status := counts.v4StatusCode
			if status == 0 {
				status = http.StatusOK
			}
			if status == http.StatusOK && counts.lastV4ETag != "" {
				w.Header().Set("X-Poll-ETag", counts.lastV4ETag)
			}
			w.WriteHeader(status)
			if status == http.StatusOK {
				_, _ = w.Write(v4Body)
			}
		case "/v3/configs/config/secrets/download":
			counts.v3Download++
			if counts.v3RespETag != "" {
				w.Header().Set("etag", counts.v3RespETag)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(v3Body)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func fetchTestScopedOptions(host string) models.ScopedOptions {
	return models.ScopedOptions{
		Token:          models.ScopedOption{Value: "tok"},
		APIHost:        models.ScopedOption{Value: host},
		VerifyTLS:      models.ScopedOption{Value: "true"},
		EnclaveProject: models.ScopedOption{Value: fetchTestProject},
		EnclaveConfig:  models.ScopedOption{Value: fetchTestConfig},
	}
}

func fetchTestFallbackOpts(path string) FallbackOptions {
	return FallbackOptions{
		Enable:             true,
		Path:               path,
		Readonly:           false,
		Exclusive:          false,
		ExitOnWriteFailure: false,
		Passphrase:         fetchTestPassphrase,
	}
}

// writeFetchFallbackFile encrypts and writes a fallback file and returns its raw (encrypted) hash.
func writeFetchFallbackFile(t *testing.T, path string, plaintext []byte) string {
	t.Helper()
	encrypted, err := crypto.Encrypt(fetchTestPassphrase, plaintext, "base64")
	if err != nil {
		t.Fatalf("failed to encrypt fallback file: %v", err)
	}
	if err := os.WriteFile(path, []byte(encrypted), 0600); err != nil {
		t.Fatalf("failed to write fallback file: %v", err)
	}
	return crypto.Hash(encrypted)
}

// TestFetchSecretsPollCurrentCacheHit: valid PollETag + matching identity + valid hash,
// poll returns current => return decrypted fallback bytes, from cache = true, no download hit.
func TestFetchSecretsPollCurrentCacheHit(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	cachedSecrets := []byte(`{"CACHED":"true"}`)
	hash := writeFetchFallbackFile(t, fallbackPath, cachedSecrets)
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	counts := &endpointCounts{}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.True(t, fromCache, "expected from cache = true")
	assert.Equal(t, cachedSecrets, response, "expected decrypted cached secrets")
	assert.Equal(t, 1, counts.poll, "expected poll to be hit once")
	assert.Equal(t, 0, counts.v4Download, "v4 download must not be hit")
	assert.Equal(t, 0, counts.v3Download, "v3 download must not be hit")
}

// TestFetchSecretsPollChangedV4Download: poll changed => v4 200 with fresh X-Poll-ETag =>
// return v4 body, from cache = false, metadata updated with new PollETag + identity.
func TestFetchSecretsPollChangedV4Download(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	newV4Body := []byte(`{"V4NEW":"yes"}`)
	newPollETag := "new-poll-etag-xyz"
	counts := &endpointCounts{pollStatus: http.StatusOK, lastV4ETag: newPollETag}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), newV4Body)
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false")
	assert.Equal(t, newV4Body, response, "expected v4 response body")
	assert.Equal(t, 1, counts.poll)
	assert.Equal(t, 1, counts.v4Download)
	assert.Equal(t, 0, counts.v3Download, "v3 must not be hit")

	metadata, err := MetadataFile(metadataPath)
	assert.True(t, err.IsNil(), "metadata should be readable")
	assert.Equal(t, newPollETag, metadata.PollETag, "PollETag should be updated to v4 header")
	assert.Equal(t, identity, metadata.RequestIdentity, "RequestIdentity should match this invocation")
}

// TestFetchSecretsPoll503V3Fallback: poll 503 (PollUnavailable) => v3 flow this invocation,
// metadata byte-for-byte unchanged (v3 returns no etag header).
func TestFetchSecretsPoll503V3Fallback(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}
	metadataBefore, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("failed to read metadata: %v", readErr)
	}

	v3Body := []byte(`{"V3":"fallback"}`)
	counts := &endpointCounts{pollStatus: http.StatusServiceUnavailable}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false (v3 200 body)")
	assert.Equal(t, v3Body, response, "expected v3 body")
	assert.Equal(t, 1, counts.poll)
	assert.Equal(t, 0, counts.v4Download, "v4 must not be hit on PollUnavailable")
	assert.Equal(t, 1, counts.v3Download, "v3 must be hit")

	metadataAfter, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("failed to read metadata: %v", readErr)
	}
	assert.Equal(t, metadataBefore, metadataAfter, "metadata must be unchanged after PollUnavailable -> v3 (no etag)")
}

// TestFetchSecretsPoll404NoMarker: poll 404 (PollUnavailable) => v3 flow; and a second
// invocation still probes the poll endpoint again (no persisted marker).
func TestFetchSecretsPoll404NoMarker(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	v3Body := []byte(`{"V3":"fallback"}`)
	counts := &endpointCounts{pollStatus: http.StatusNotFound}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	// readonly so the v3 fallback path doesn't rewrite the fallback file (which would invalidate
	// the stored hash and change which ladder bucket the second invocation lands in). This keeps
	// the stored poll context intact so we can assert the second invocation re-probes the poll
	// endpoint rather than being permanently downgraded to v3-only.
	fallbackOpts := fetchTestFallbackOpts(fallbackPath)
	fallbackOpts.Readonly = true

	_, _ = FetchSecrets(localConfig, true, fallbackOpts, metadataPath, nil, 0, models.JSON, nil)
	assert.Equal(t, 1, counts.poll, "first invocation should probe poll")
	assert.Equal(t, 1, counts.v3Download)

	// second invocation with same state must probe poll again (no persisted marker)
	_, _ = FetchSecrets(localConfig, true, fallbackOpts, metadataPath, nil, 0, models.JSON, nil)
	assert.Equal(t, 2, counts.poll, "second invocation must probe poll again (no marker)")
	assert.Equal(t, 0, counts.v4Download)
}

// TestFetchSecretsIdentityMismatchStraightToV4: metadata has valid PollETag but a
// RequestIdentity that doesn't match => skip poll, go straight to v4 download.
func TestFetchSecretsIdentityMismatchStraightToV4(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	// identity for a DIFFERENT project => mismatch with the call below
	otherIdentity := SecretsRequestIdentity("other-project", fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, otherIdentity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	v4Body := []byte(`{"V4":"direct"}`)
	counts := &endpointCounts{lastV4ETag: "fresh-poll-etag"}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), v4Body)
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache)
	assert.Equal(t, v4Body, response)
	assert.Equal(t, 0, counts.poll, "poll must not be hit on identity mismatch")
	assert.Equal(t, 1, counts.v4Download, "v4 download must be hit directly")
	assert.Equal(t, 0, counts.v3Download)
}

// TestFetchSecretsNoCacheV3Only: enableCache = false => neither poll nor v4 hit; only v3.
func TestFetchSecretsNoCacheV3Only(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	v3Body := []byte(`{"V3":"nocache"}`)
	counts := &endpointCounts{}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, false, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache)
	assert.Equal(t, v3Body, response)
	assert.Equal(t, 0, counts.poll, "poll must not be hit with cache disabled")
	assert.Equal(t, 0, counts.v4Download, "v4 must not be hit with cache disabled")
	assert.Equal(t, 1, counts.v3Download, "only v3 must be hit")
}

// TestFetchSecretsV4Download404FirstFetch: no metadata (first fetch) => v4 download 404 =>
// v3 flow succeeds, no v4-success artifacts leaked into metadata.
func TestFetchSecretsV4Download404FirstFetch(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	v3Body := []byte(`{"V3":"firstfetch"}`)
	// v3 returns no etag header => v3 success path writes no metadata
	counts := &endpointCounts{v4StatusCode: http.StatusNotFound}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache)
	assert.Equal(t, v3Body, response)
	assert.Equal(t, 0, counts.poll, "poll must not be hit (no stored PollETag)")
	assert.Equal(t, 1, counts.v4Download, "v4 download probed first")
	assert.Equal(t, 1, counts.v3Download, "v3 flow used after v4 404")

	// no v4-success artifacts: either no metadata file, or empty PollETag/RequestIdentity
	if _, statErr := os.Stat(metadataPath); statErr == nil {
		metadata, err := MetadataFile(metadataPath)
		assert.True(t, err.IsNil())
		assert.Equal(t, "", metadata.PollETag, "no v4 poll etag should leak into metadata")
		assert.Equal(t, "", metadata.RequestIdentity, "no v4 request identity should leak into metadata")
	}
}

// TestFetchSecretsPoll503V3FallbackWithETagPreservesPollETag: poll 503 (PollUnavailable) => v3
// flow this invocation, and the v3 server DOES return an ETag header (the case the plain 503 test
// above does not exercise). Because the pre-seeded RequestIdentity matches this invocation, the
// v3-200 metadata write must PRESERVE the existing PollETag rather than clobbering it to "".
func TestFetchSecretsPoll503V3FallbackWithETagPreservesPollETag(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	v3Body := []byte(`{"V3":"fallback"}`)
	v3ETag := "v3-etag-def456"
	counts := &endpointCounts{pollStatus: http.StatusServiceUnavailable, v3RespETag: v3ETag}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false (v3 200 body)")
	assert.Equal(t, v3Body, response, "expected v3 body")
	assert.Equal(t, 1, counts.poll)
	assert.Equal(t, 0, counts.v4Download, "v4 must not be hit on PollUnavailable")
	assert.Equal(t, 1, counts.v3Download, "v3 must be hit")

	metadata, err := MetadataFile(metadataPath)
	assert.True(t, err.IsNil(), "metadata should be readable")
	assert.Equal(t, fetchTestPollETag, metadata.PollETag, "existing PollETag must be preserved (identity matches)")
	assert.Equal(t, v3ETag, metadata.ETag, "v3 ETag must be updated")
	assert.Equal(t, identity, metadata.RequestIdentity, "RequestIdentity must remain this invocation's identity")
}

// TestFetchSecretsPoll503V3FallbackWithETagIdentityMismatchClearsPollETag: same as above but the
// pre-seeded RequestIdentity does NOT match this invocation. The v3-200 metadata write must clear
// the PollETag to "" (today's behavior for the mismatch case), because the stored poll context
// belongs to a different request shape and must not be carried forward.
func TestFetchSecretsPoll503V3FallbackWithETagIdentityMismatchClearsPollETag(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	// identity for a DIFFERENT project => mismatch with the call below. A non-matching identity
	// with a non-empty PollETag means validPollETag returns "" (no poll attempt), so this
	// invocation goes v4-download-first; force v4 to fail so it falls through to v3.
	otherIdentity := SecretsRequestIdentity("other-project", fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, otherIdentity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	v3Body := []byte(`{"V3":"fallback"}`)
	v3ETag := "v3-etag-def456"
	counts := &endpointCounts{v4StatusCode: http.StatusNotFound, v3RespETag: v3ETag}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false (v3 200 body)")
	assert.Equal(t, v3Body, response, "expected v3 body")
	assert.Equal(t, 0, counts.poll, "poll must not be hit on identity mismatch")
	assert.Equal(t, 1, counts.v4Download, "v4 download probed first")
	assert.Equal(t, 1, counts.v3Download, "v3 must be hit after v4 failure")

	currentIdentity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	metadata, err := MetadataFile(metadataPath)
	assert.True(t, err.IsNil(), "metadata should be readable")
	assert.Equal(t, "", metadata.PollETag, "PollETag must be cleared on identity mismatch")
	assert.Equal(t, v3ETag, metadata.ETag, "v3 ETag must be updated")
	assert.Equal(t, currentIdentity, metadata.RequestIdentity, "RequestIdentity must be updated to this invocation's identity")
}

// TestFetchSecretsPoll503V3FallbackRejectedEtagNotReadmitted: metadata is pre-seeded with a
// matching RequestIdentity and a non-empty PollETag, but an EMPTY Hash. validPollETag fails this
// closed (see TestValidPollETagEmptyHashFailsClosed) and returns "", so no poll is attempted; this
// invocation goes v4-download-first, v4 fails, and falls through to v3, which returns 200 with an
// ETag header. The v3-write path must NOT re-admit the rejected PollETag by re-reading it from
// disk: since validPollETag never validated it this invocation (attemptedPollETag == ""), the
// written metadata must have PollETag "".
func TestFetchSecretsPoll503V3FallbackRejectedEtagNotReadmitted(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	// Write the fallback file so a hash COULD be computed, but seed metadata with an empty Hash
	// alongside a non-empty PollETag -- exactly the "rejected" shape from
	// TestValidPollETagEmptyHashFailsClosed.
	writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", "", fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	// Sanity-check the precondition this test relies on: validPollETag rejects this metadata.
	if got := validPollETag(metadataPath, fallbackPath, identity); got != "" {
		t.Fatalf("test precondition failed: expected validPollETag to reject empty-Hash metadata, got %q", got)
	}

	v3Body := []byte(`{"V3":"fallback"}`)
	v3ETag := "v3-etag-after-rejection"
	// v4 download fails (404) so this invocation falls through to v3.
	counts := &endpointCounts{v4StatusCode: http.StatusNotFound, v3RespETag: v3ETag}
	server := newFetchTestServer(t, counts, v3Body, []byte(`{"V4":"1"}`))
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false (v3 200 body)")
	assert.Equal(t, v3Body, response, "expected v3 body")
	assert.Equal(t, 0, counts.poll, "poll must not be hit when validPollETag rejects stored metadata")
	assert.Equal(t, 1, counts.v4Download, "v4 download probed first")
	assert.Equal(t, 1, counts.v3Download, "v3 must be hit after v4 failure")

	metadata, err := MetadataFile(metadataPath)
	assert.True(t, err.IsNil(), "metadata should be readable")
	assert.Equal(t, "", metadata.PollETag, "rejected PollETag must NOT be re-admitted into fresh, valid metadata")
	assert.Equal(t, v3ETag, metadata.ETag, "v3 ETag must be updated")
	assert.NotEqual(t, "", metadata.Hash, "v3 write produces a fresh, valid hash")
	assert.Equal(t, identity, metadata.RequestIdentity, "RequestIdentity must match this invocation")
}

// TestValidPollETagEmptyHashFailsClosed: metadata with a non-empty PollETag but an empty Hash
// (which no legitimate writer produces) must fail closed and return "" rather than tolerating
// the missing hash, since a valid PollETag write always includes its hash.
func TestValidPollETagEmptyHashFailsClosed(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", "", fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	assert.Equal(t, "", validPollETag(metadataPath, fallbackPath, identity), "empty Hash must fail closed, never serve the cache via the poll path")
}

// TestFetchSecretsPollChangedV4ReadonlyNoWrites: metadata + fallback pre-seeded, poll reports
// changed, and fallbackOpts.Readonly = true. The v4 200 response must still be returned to the
// caller, but downloadSecretsV4 must not touch disk at all: the fallback file and metadata file
// must be byte-for-byte unchanged (Readonly disables the writeFallbackFile gate in
// downloadSecretsV4, same as it does in the v3 write path).
func TestFetchSecretsPollChangedV4ReadonlyNoWrites(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	hash := writeFetchFallbackFile(t, fallbackPath, []byte(`{"OLD":"1"}`))
	identity := SecretsRequestIdentity(fetchTestProject, fetchTestConfig, models.JSON, nil, 0, nil)
	if err := WriteMetadataFile(metadataPath, "", hash, fetchTestPollETag, identity); !err.IsNil() {
		t.Fatalf("failed to write metadata: %+v", err)
	}

	fallbackBefore, readErr := os.ReadFile(fallbackPath)
	if readErr != nil {
		t.Fatalf("failed to read fallback file: %v", readErr)
	}
	metadataBefore, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("failed to read metadata: %v", readErr)
	}

	newV4Body := []byte(`{"V4NEW":"yes"}`)
	counts := &endpointCounts{pollStatus: http.StatusOK, lastV4ETag: "new-poll-etag-xyz"}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), newV4Body)
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	fallbackOpts := fetchTestFallbackOpts(fallbackPath)
	fallbackOpts.Readonly = true

	response, fromCache := FetchSecrets(localConfig, true, fallbackOpts, metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false")
	assert.Equal(t, newV4Body, response, "expected v4 response body returned even in readonly mode")
	assert.Equal(t, 1, counts.poll)
	assert.Equal(t, 1, counts.v4Download)
	assert.Equal(t, 0, counts.v3Download, "v3 must not be hit")

	fallbackAfter, readErr := os.ReadFile(fallbackPath)
	if readErr != nil {
		t.Fatalf("failed to read fallback file: %v", readErr)
	}
	metadataAfter, readErr := os.ReadFile(metadataPath)
	if readErr != nil {
		t.Fatalf("failed to read metadata: %v", readErr)
	}
	assert.Equal(t, fallbackBefore, fallbackAfter, "fallback file must be byte-unchanged in readonly mode")
	assert.Equal(t, metadataBefore, metadataAfter, "metadata file must be byte-unchanged in readonly mode")
}

// TestFetchSecretsFirstFetchV4ReadonlyNoFilesCreated: no cache/metadata at all (first fetch) and
// fallbackOpts.Readonly = true. The v4 200 response must be returned, but no fallback file or
// metadata file may be created on disk.
func TestFetchSecretsFirstFetchV4ReadonlyNoFilesCreated(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	v4Body := []byte(`{"V4":"firstfetch-readonly"}`)
	counts := &endpointCounts{lastV4ETag: "fresh-poll-etag"}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), v4Body)
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	fallbackOpts := fetchTestFallbackOpts(fallbackPath)
	fallbackOpts.Readonly = true

	response, fromCache := FetchSecrets(localConfig, true, fallbackOpts, metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache, "expected from cache = false")
	assert.Equal(t, v4Body, response, "expected v4 response body returned even in readonly mode")
	assert.Equal(t, 0, counts.poll, "poll must not be hit (no stored PollETag)")
	assert.Equal(t, 1, counts.v4Download, "v4 download probed first")
	assert.Equal(t, 0, counts.v3Download)

	if _, statErr := os.Stat(fallbackPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no fallback file to be created in readonly mode, stat err: %v", statErr)
	}
	if _, statErr := os.Stat(metadataPath); !os.IsNotExist(statErr) {
		t.Fatalf("expected no metadata file to be created in readonly mode, stat err: %v", statErr)
	}
}

// TestFetchSecretsV4DownloadEmptyPollETag: no metadata (first fetch) => v4 download 200 WITHOUT
// an X-Poll-ETag header (dynamic-lease case, e.g. a config with dynamic secrets where the server
// intentionally omits the poll etag) => metadata is written with an empty PollETag, and a second
// invocation must not attempt to poll (no valid stored poll context) and must go v4-download-first
// again.
func TestFetchSecretsV4DownloadEmptyPollETag(t *testing.T) {
	dir := t.TempDir()
	fallbackPath := filepath.Join(dir, "fallback")
	metadataPath := filepath.Join(dir, "metadata")

	v4Body := []byte(`{"V4":"dynamic-lease"}`)
	// lastV4ETag left empty => v4 download responds 200 without X-Poll-ETag header
	counts := &endpointCounts{}
	server := newFetchTestServer(t, counts, []byte(`{"V3":"1"}`), v4Body)
	defer server.Close()

	localConfig := fetchTestScopedOptions(server.URL)
	response, fromCache := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache)
	assert.Equal(t, v4Body, response, "expected v4 response body")
	assert.Equal(t, 0, counts.poll, "poll must not be hit (no stored PollETag)")
	assert.Equal(t, 1, counts.v4Download, "v4 download probed first")
	assert.Equal(t, 0, counts.v3Download, "v3 must not be hit")

	metadata, err := MetadataFile(metadataPath)
	assert.True(t, err.IsNil(), "metadata should be readable")
	assert.Equal(t, "", metadata.PollETag, "poll_etag should be empty on disk")

	// second invocation: with no valid stored poll context, must not poll and must go
	// v4-download-first again
	response2, fromCache2 := FetchSecrets(localConfig, true, fetchTestFallbackOpts(fallbackPath), metadataPath, nil, 0, models.JSON, nil)

	assert.False(t, fromCache2)
	assert.Equal(t, v4Body, response2, "expected v4 response body again")
	assert.Equal(t, 0, counts.poll, "poll must not be hit on second invocation either")
	assert.Equal(t, 2, counts.v4Download, "second invocation must go v4-download-first again")
	assert.Equal(t, 0, counts.v3Download, "v3 must not be hit")
}
