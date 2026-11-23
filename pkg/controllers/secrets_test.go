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

package controllers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type dangerousSecretNameTestCase struct {
	name                         string
	secrets                      map[string]string
	expectedDangerousSecretNames []string
}

func TestCheckForDangerousSecretNames(t *testing.T) {
	testCases := []dangerousSecretNameTestCase{
		{
			name: "Should not find any dangerous secret names",
			secrets: map[string]string{
				"MY_SECRET": "123",
			},
			expectedDangerousSecretNames: nil,
		},
		{
			name: "Should find a dangerous secret name",
			secrets: map[string]string{
				"DYLD_INSERT_LIBRARIES": "123",
				"MY_SECRET":             "123",
			},
			expectedDangerousSecretNames: []string{"DYLD_INSERT_LIBRARIES"},
		},
		{
			name: "Should find multiple dangerous secret names",
			secrets: map[string]string{
				"DYLD_INSERT_LIBRARIES": "123",
				"MY_SECRET":             "123",
				"LD_LIBRARY_PATH":       "123",
				"WINDIR":                "123",
				"PROMPT_COMMAND":        "123",
			},
			expectedDangerousSecretNames: []string{"DYLD_INSERT_LIBRARIES", "LD_LIBRARY_PATH", "WINDIR", "PROMPT_COMMAND"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := CheckForDangerousSecretNames(testCase.secrets)
			if testCase.expectedDangerousSecretNames == nil {
				assert.Nil(t, nil, err)
			} else {
				for _, v := range testCase.expectedDangerousSecretNames {
					assert.Contains(t, err.Error(), v)
				}
			}
		})
	}
}

func TestMountSecrets(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	secrets := []byte(`{"SECRET_KEY":"secret_value"}`)
	mountPath := filepath.Join(t.TempDir(), "secrets_mount")

	path, cleanup, err := MountSecrets(secrets, mountPath, 1)
	if !err.IsNil() {
		t.Fatalf("MountSecrets failed: %v", err.Err)
	}
	defer cleanup()

	time.Sleep(50 * time.Millisecond)

	content, readErr := os.ReadFile(path)
	assert.NoError(t, readErr)
	assert.Equal(t, string(secrets), string(content))
}

func TestMountSecretsBrokenPipe(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	// large data to exceed pipe buffer and trigger EPIPE when reader closes early
	secrets := make([]byte, 256*1024)
	for i := range secrets {
		secrets[i] = byte('A' + (i % 26))
	}
	mountPath := filepath.Join(t.TempDir(), "secrets_mount_epipe")

	path, cleanup, err := MountSecrets(secrets, mountPath, 11)
	if !err.IsNil() {
		t.Fatalf("MountSecrets failed: %v", err.Err)
	}
	defer cleanup()

	time.Sleep(50 * time.Millisecond)

	for i := 0; i < 10; i++ {
		f, openErr := os.OpenFile(path, os.O_RDONLY, 0)
		if openErr != nil {
			continue
		}
		f.Read(make([]byte, 1))
		f.Close()
		time.Sleep(5 * time.Millisecond)
	}

	content, readErr := os.ReadFile(path)
	assert.NoError(t, readErr, "mount should survive broken pipe")
	assert.Equal(t, secrets, content)
}

func TestPrepareSecretsMountsRawBytes(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	// Raw bytes from the backend (simulating what FetchSecrets returns)
	// This tests that PrepareSecrets uses the raw bytes directly for mounting
	secretsBytes := []byte("SECRET=value\\nwith_escaped_newline")

	mountPath := filepath.Join(t.TempDir(), "formatted_secrets")
	mountOptions := MountOptions{
		Enable:   true,
		Format:   "docker",
		Path:     mountPath,
		MaxReads: 1,
	}

	// Pass empty secrets map and raw bytes for mounting
	env, cleanup := PrepareSecrets(map[string]string{}, secretsBytes, []string{}, "false", mountOptions)
	if cleanup != nil {
		defer cleanup()
	}

	assert.Equal(t, string(secretsBytes), readMountedSecrets(t, env), "mounted file should contain raw bytes from backend")
}

func readMountedSecrets(t *testing.T, env []string) string {
	t.Helper()
	var secretsPath string
	for _, e := range env {
		if strings.HasPrefix(e, "DOPPLER_CLI_SECRETS_PATH=") {
			secretsPath = strings.TrimPrefix(e, "DOPPLER_CLI_SECRETS_PATH=")
		}
	}
	if !assert.NotEmpty(t, secretsPath, "DOPPLER_CLI_SECRETS_PATH should be set") {
		return ""
	}
	time.Sleep(50 * time.Millisecond)
	content, err := os.ReadFile(secretsPath)
	assert.NoError(t, err)
	return string(content)
}

func TestPrepareSecretsEnvInjectionPrecedence(t *testing.T) {
	originalEnv := []string{"FOO=env-foo", "BAR=env-bar", "UNRELATED=env-unrelated", "HOME=env-home"}
	testCases := []struct {
		preserveEnv string
		expected    []string
	}{
		{"false", []string{"FOO=doppler-foo", "BAR=doppler-bar", "UNRELATED=env-unrelated", "HOME=env-home", "ONLY_DOPPLER=doppler-only"}},
		{"true", []string{"FOO=env-foo", "BAR=env-bar", "UNRELATED=env-unrelated", "HOME=env-home", "ONLY_DOPPLER=doppler-only"}},
		{"FOO", []string{"FOO=env-foo", "BAR=doppler-bar", "UNRELATED=env-unrelated", "HOME=env-home", "ONLY_DOPPLER=doppler-only"}},
		{" FOO, ,BAR ", []string{"FOO=env-foo", "BAR=env-bar", "UNRELATED=env-unrelated", "HOME=env-home", "ONLY_DOPPLER=doppler-only"}},
	}

	for _, tc := range testCases {
		t.Run(tc.preserveEnv, func(t *testing.T) {
			dopplerSecrets := map[string]string{"FOO": "doppler-foo", "BAR": "doppler-bar", "HOME": "doppler-home", "ONLY_DOPPLER": "doppler-only"}
			env, _ := PrepareSecrets(dopplerSecrets, nil, originalEnv, tc.preserveEnv, MountOptions{})
			assert.ElementsMatch(t, tc.expected, env)
		})
	}
}

func TestPrepareSecretsMountJSONPreserveEnv(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	secretsBytes := []byte(`{"BAR":"doppler-bar","FOO":"doppler-foo","HOME":"doppler-home"}`)
	originalEnv := []string{"FOO=<env&foo>", "UNRELATED=env-unrelated", "HOME=env-home", "OTHER=env-other"}

	testCases := []struct {
		name        string
		preserveEnv string
		expected    string
	}{
		{"disabled mounts backend bytes", "false", string(secretsBytes)},
		{"true merges the whole env like env injection", "true", `{"BAR":"doppler-bar","FOO":"<env&foo>","HOME":"env-home","OTHER":"env-other","UNRELATED":"env-unrelated"}`},
		{"list overrides only listed Doppler secrets", "FOO", `{"BAR":"doppler-bar","FOO":"<env&foo>","HOME":"doppler-home"}`},
		{"list adds listed env vars that aren't Doppler secrets", "FOO,UNRELATED", `{"BAR":"doppler-bar","FOO":"<env&foo>","HOME":"doppler-home","UNRELATED":"env-unrelated"}`},
		{"listed name missing from env keeps Doppler value", "BAR,MISSING", `{"BAR":"doppler-bar","FOO":"doppler-foo","HOME":"doppler-home"}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dopplerSecrets := map[string]string{"FOO": "doppler-foo", "BAR": "doppler-bar", "HOME": "doppler-home"}
			mountOptions := MountOptions{Enable: true, Format: "json", Path: filepath.Join(t.TempDir(), "secrets.json"), MaxReads: 1}
			env, cleanup := PrepareSecrets(dopplerSecrets, secretsBytes, originalEnv, tc.preserveEnv, mountOptions)
			if cleanup != nil {
				defer cleanup()
			}
			assert.Equal(t, tc.expected, readMountedSecrets(t, env))
			assert.Equal(t, "doppler-foo", dopplerSecrets["FOO"], "caller's map must not be mutated")
		})
	}
}

func TestPrepareSecretsMountTemplatePreserveEnv(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	dopplerSecrets := map[string]string{"FOO": "doppler-foo", "BAR": "doppler-bar", "PS1": "doppler-ps1"}
	mountOptions := MountOptions{
		Enable:   true,
		Format:   "template",
		Path:     filepath.Join(t.TempDir(), "secrets.txt"),
		Template: "{{.FOO}} {{.BAR}} {{.UNRELATED}} {{.PS1}}",
		MaxReads: 1,
	}
	env, cleanup := PrepareSecrets(dopplerSecrets, nil, []string{"FOO=env-foo", "UNRELATED=env-unrelated"}, "true", mountOptions)
	if cleanup != nil {
		defer cleanup()
	}
	assert.Equal(t, "env-foo doppler-bar env-unrelated doppler-ps1", readMountedSecrets(t, env))
}

func TestPrepareSecretsMountOnlySecretsLimitsPreserveEnv(t *testing.T) {
	if !utils.SupportsNamedPipes {
		t.Skip("Named pipes not supported on this platform")
	}

	for _, preserveEnv := range []string{"true", "FOO,BAR"} {
		t.Run(preserveEnv, func(t *testing.T) {
			mountOptions := MountOptions{
				Enable:           true,
				Format:           "json",
				Path:             filepath.Join(t.TempDir(), "secrets.json"),
				MaxReads:         1,
				SecretsToInclude: []string{"FOO"},
			}
			env, cleanup := PrepareSecrets(map[string]string{"FOO": "doppler-foo"}, nil, []string{"FOO=env-foo", "BAR=env-bar"}, preserveEnv, mountOptions)
			if cleanup != nil {
				defer cleanup()
			}
			assert.Equal(t, `{"FOO":"env-foo"}`, readMountedSecrets(t, env))
		})
	}
}
