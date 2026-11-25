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
package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runAsCLIEnv makes the test binary act as the doppler CLI. Each command runs in its own
// process so cobra flag values and package globals (--json, --silent) start fresh.
const runAsCLIEnv = "DOPPLER_CMD_TEST_RUN_AS_CLI"

func TestMain(m *testing.M) {
	if os.Getenv(runAsCLIEnv) == "1" {
		args := os.Args[1:]
		for i, arg := range args {
			if arg == "--" {
				args = args[i+1:]
				break
			}
		}
		rootCmd.SetArgs(args)
		Execute()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

const (
	valueA    = "value-a-should-not-print"
	rawValueB = "raw-b-should-not-print-${A}"
	valueB    = "computed-b-should-not-print"
)

// fakeAPI answers secret writes with the full config, as the real API does: SECRET_B is
// in the response even though only SECRET_A was written
func fakeAPI(t *testing.T) (*httptest.Server, *[]map[string]interface{}) {
	t.Helper()
	var writes []map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/configs/config/secrets", "/v3/configs/config/secrets/upload":
			if r.Method == http.MethodPost {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("reading request body: %v", err)
				}
				var parsed map[string]interface{}
				if err := json.Unmarshal(body, &parsed); err != nil {
					t.Errorf("parsing request body: %v", err)
				}
				parsed["path"] = r.URL.Path
				writes = append(writes, parsed)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"secrets":{
				"SECRET_A":{"raw":"` + valueA + `","computed":"` + valueA + `","rawVisibility":"masked","computedVisibility":"masked","note":""},
				"SECRET_B":{"raw":"` + rawValueB + `","computed":"` + valueB + `","rawVisibility":"masked","computedVisibility":"masked","note":""}
			}}`))
		default:
			// analytics and anything else the CLI calls on its own
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true}`))
		}
	}))
	t.Cleanup(server.Close)
	return server, &writes
}

func runCLI(t *testing.T, server *httptest.Server, args ...string) (string, string) {
	t.Helper()
	home := t.TempDir()
	base := []string{
		"--no-read-env",
		"--no-check-version",
		"--config-dir", filepath.Join(home, ".doppler"),
		"--api-host", server.URL,
		"--token", "dp.st.fake",
	}
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=^$", "--"}, append(base, args...)...)...) // #nosec G204
	cmd.Env = []string{runAsCLIEnv + "=1", "HOME=" + home, "PATH=" + os.Getenv("PATH")}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("doppler %s: %v\nstdout: %s\nstderr: %s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String()
}

func assertNoValues(t *testing.T, outputs ...string) {
	t.Helper()
	for _, out := range outputs {
		for _, value := range []string{valueA, rawValueB, valueB} {
			if strings.Contains(out, value) {
				t.Errorf("output contains secret value %q:\n%s", value, out)
			}
		}
	}
}

func parseJSONSecrets(t *testing.T, stdout string) map[string]map[string]interface{} {
	t.Helper()
	var parsed map[string]map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &parsed); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	return parsed
}

func writeUploadFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "upload.env")
	if err := os.WriteFile(path, []byte("SECRET_A="+valueA+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDeleteSecretsPrintsSummary(t *testing.T) {
	server, writes := fakeAPI(t)

	stdout, stderr := runCLI(t, server, "secrets", "delete", "SECRET_C", "SECRET_D", "--yes", "-p", "backend", "-c", "dev")

	if stdout != "" {
		t.Errorf("expected no stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Deleted SECRET_C, SECRET_D in backend/dev") {
		t.Errorf("expected summary on stderr, got:\n%s", stderr)
	}
	assertNoValues(t, stdout, stderr)

	if len(*writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(*writes))
	}
	sent, _ := (*writes)[0]["secrets"].(map[string]interface{})
	if len(sent) != 2 || sent["SECRET_C"] != nil || sent["SECRET_D"] != nil {
		t.Errorf("expected SECRET_C and SECRET_D to be sent as null, got %v", (*writes)[0]["secrets"])
	}
}

func TestRawWithoutJSONPrintsNoValues(t *testing.T) {
	uploadFile := writeUploadFile(t)
	tests := []struct {
		name string
		args []string
	}{
		{"secrets", []string{"secrets", "--raw"}},
		{"secrets get", []string{"secrets", "get", "SECRET_A", "SECRET_B", "--raw"}},
		{"secrets set", []string{"secrets", "set", "SECRET_A=new-value", "--raw"}},
		{"secrets delete", []string{"secrets", "delete", "SECRET_C", "--yes", "--raw"}},
		{"secrets upload", []string{"secrets", "upload", uploadFile, "--raw"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := fakeAPI(t)

			stdout, stderr := runCLI(t, server, tt.args...)

			if strings.Contains(stderr, "deprecated") {
				t.Errorf("--raw should not be reported as deprecated:\n%s", stderr)
			}
			assertNoValues(t, stdout, stderr)
		})
	}
}

func TestDeleteSecretsSilent(t *testing.T) {
	server, writes := fakeAPI(t)

	stdout, stderr := runCLI(t, server, "secrets", "delete", "SECRET_C", "--yes", "--silent")

	if stdout != "" || stderr != "" {
		t.Errorf("expected no output with --silent, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
	if len(*writes) != 1 {
		t.Errorf("expected 1 write, got %d", len(*writes))
	}
}

func TestDeleteSecretsJSON(t *testing.T) {
	server, _ := fakeAPI(t)

	stdout, _ := runCLI(t, server, "secrets", "delete", "SECRET_C", "--yes", "--json")
	secrets := parseJSONSecrets(t, stdout)
	if len(secrets) != 2 {
		t.Fatalf("expected the config's 2 secrets, got %d:\n%s", len(secrets), stdout)
	}
	if secrets["SECRET_B"]["computed"] != valueB {
		t.Errorf("expected computed value for SECRET_B, got %v", secrets["SECRET_B"]["computed"])
	}
	if _, ok := secrets["SECRET_B"]["raw"]; ok {
		t.Errorf("expected no raw value without --raw, got %v", secrets["SECRET_B"])
	}

	stdout, _ = runCLI(t, server, "secrets", "delete", "SECRET_C", "--yes", "--json", "--raw")
	secrets = parseJSONSecrets(t, stdout)
	if secrets["SECRET_B"]["raw"] != rawValueB {
		t.Errorf("expected raw value with --raw, got %v", secrets["SECRET_B"]["raw"])
	}
}

func TestUploadSecretsPrintsSummary(t *testing.T) {
	server, writes := fakeAPI(t)
	path := writeUploadFile(t)

	stdout, stderr := runCLI(t, server, "secrets", "upload", path, "-p", "backend", "-c", "dev")

	if stdout != "" {
		t.Errorf("expected no stdout, got:\n%s", stdout)
	}
	if !strings.Contains(stderr, "Uploaded secrets from "+path+" in backend/dev") {
		t.Errorf("expected summary on stderr, got:\n%s", stderr)
	}
	assertNoValues(t, stdout, stderr)

	if len(*writes) != 1 || (*writes)[0]["path"] != "/v3/configs/config/secrets/upload" {
		t.Fatalf("expected 1 upload, got %v", *writes)
	}
}

func TestUploadSecretsSummaryWithoutLocalConfig(t *testing.T) {
	server, _ := fakeAPI(t)
	path := writeUploadFile(t)

	// with a service token, project and config come from the token, so there's nothing to name
	_, stderr := runCLI(t, server, "secrets", "upload", path)

	if !strings.Contains(stderr, "Uploaded secrets from "+path+"\n") {
		t.Errorf("expected summary without a destination, got:\n%s", stderr)
	}
}

func TestUploadSecretsSilent(t *testing.T) {
	server, _ := fakeAPI(t)

	stdout, stderr := runCLI(t, server, "secrets", "upload", writeUploadFile(t), "--silent")

	if stdout != "" || stderr != "" {
		t.Errorf("expected no output with --silent, got stdout:\n%s\nstderr:\n%s", stdout, stderr)
	}
}

func TestUploadSecretsJSON(t *testing.T) {
	server, _ := fakeAPI(t)

	stdout, stderr := runCLI(t, server, "secrets", "upload", writeUploadFile(t), "--json", "--raw")

	if strings.Contains(stderr, "Uploaded") {
		t.Errorf("expected no summary in --json mode, got:\n%s", stderr)
	}
	secrets := parseJSONSecrets(t, stdout)
	if len(secrets) != 2 {
		t.Fatalf("expected the config's 2 secrets, got %d:\n%s", len(secrets), stdout)
	}
	if secrets["SECRET_A"]["computed"] != valueA || secrets["SECRET_B"]["raw"] != rawValueB {
		t.Errorf("expected computed and raw values, got %v", secrets)
	}
}
