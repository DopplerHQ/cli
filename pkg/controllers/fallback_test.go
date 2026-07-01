/*
Copyright © 2022 Doppler <support@doppler.com>

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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/stretchr/testify/assert"
)

type fallbackFileHashTestCase struct {
	name          string
	secretNamesA  []string
	secretNamesB  []string
	expectedEqual bool
}

func TestGenerateFallbackFileHash(t *testing.T) {
	testCases := []fallbackFileHashTestCase{
		{
			name:          "unique hash per secret names",
			secretNamesA:  []string{"A"},
			secretNamesB:  []string{"B"},
			expectedEqual: false,
		},
		{
			name:          "sort secret names",
			secretNamesA:  []string{"A", "B"},
			secretNamesB:  []string{"B", "A"},
			expectedEqual: true,
		},
		{
			name:          "dedupe secret names",
			secretNamesA:  []string{"A", "B"},
			secretNamesB:  []string{"A", "B", "B"},
			expectedEqual: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			const token = "abc"
			const project = "abc"
			const config = "abc"

			if testCase.expectedEqual {
				assert.Equal(t, GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesA), GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesB))
			} else {
				assert.NotEqual(t, GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesA), GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesB))
			}
		})

	}

}

func TestSecretsRequestIdentity(t *testing.T) {
	transformerA := &models.SecretsNameTransformer{Type: "upper-camel"}
	transformerB := &models.SecretsNameTransformer{Type: "lower-snake"}

	// stable regardless of secrets argument order
	assert.Equal(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"A", "B"}),
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"B", "A"}),
		"identity should be stable regardless of secrets order")

	// deduped secrets produce the same identity
	assert.Equal(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"A", "B"}),
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"A", "B", "B"}),
		"identity should dedupe secrets")

	// differs when format differs
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj", "cfg", models.ENV, nil, 0, nil),
		"identity should differ when format differs")

	// differs when nameTransformer differs (nil vs non-nil)
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj", "cfg", models.JSON, transformerA, 0, nil),
		"identity should differ when transformer goes from nil to non-nil")

	// differs when nameTransformer type differs
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, transformerA, 0, nil),
		SecretsRequestIdentity("proj", "cfg", models.JSON, transformerB, 0, nil),
		"identity should differ when transformer type differs")

	// differs when secrets subset differs
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"A"}),
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, []string{"A", "B"}),
		"identity should differ when secrets subset differs")

	// differs when dynamicSecretsTTL differs
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 5*time.Minute, nil),
		"identity should differ when TTL differs")

	// differs when project differs (holding config/format/etc constant) — guards against
	// two projects colliding on the same metadata path when config is empty
	assert.NotEqual(t,
		SecretsRequestIdentity("proj-a", "cfg", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj-b", "cfg", models.JSON, nil, 0, nil),
		"identity should differ when project differs")

	// differs when config differs (holding project/format/etc constant) — guards against
	// two configs colliding on the same metadata path when project is empty
	assert.NotEqual(t,
		SecretsRequestIdentity("proj", "cfg-a", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj", "cfg-b", models.JSON, nil, 0, nil),
		"identity should differ when config differs")

	// identity with empty project/config differs from identity with set project/config —
	// this is the exact collision scenario: one token, empty scoping vs. an explicit
	// project+config, must not produce the same identity
	assert.NotEqual(t,
		SecretsRequestIdentity("", "", models.JSON, nil, 0, nil),
		SecretsRequestIdentity("proj", "cfg", models.JSON, nil, 0, nil),
		"identity with empty project/config should differ from identity with set project/config")
}

func TestMetadataFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metadata-roundtrip.json")

	err := WriteMetadataFile(path, "etag-value", "hash-value", "poll-etag-value", "request-identity-value")
	assert.True(t, err.IsNil(), "WriteMetadataFile should succeed")

	metadata, readErr := MetadataFile(path)
	assert.True(t, readErr.IsNil(), "MetadataFile should succeed")

	assert.Equal(t, "1", metadata.Version)
	assert.Equal(t, "etag-value", metadata.ETag)
	assert.Equal(t, "hash-value", metadata.Hash)
	assert.Equal(t, "poll-etag-value", metadata.PollETag)
	assert.Equal(t, "request-identity-value", metadata.RequestIdentity)
}

func TestMetadataFileLegacyParsesCleanly(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".metadata-legacy.json")

	// legacy metadata: only version/etag/hash present, no poll_etag or request_identity
	legacy := []byte("version: \"1\"\netag: legacy-etag\nhash: legacy-hash\n")
	assert.NoError(t, os.WriteFile(path, legacy, 0600))

	metadata, readErr := MetadataFile(path)
	assert.True(t, readErr.IsNil(), "legacy metadata should parse without error")

	assert.Equal(t, "1", metadata.Version)
	assert.Equal(t, "legacy-etag", metadata.ETag)
	assert.Equal(t, "legacy-hash", metadata.Hash)
	assert.Equal(t, "", metadata.PollETag)
	assert.Equal(t, "", metadata.RequestIdentity)
}
