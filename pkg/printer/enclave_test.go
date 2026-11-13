package printer

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/DopplerHQ/cli/pkg/models"
)

// captureStdout runs fn and returns everything it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("unable to create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = orig
			_ = w.Close()
		}()
		fn()
	}()

	out := <-done
	_ = r.Close()
	return out
}

func strPtr(s string) *string {
	return &s
}

const (
	testSecretName     = "API_KEY"
	testComputedValue  = "cmp-s3cr3t"
	testRawValue       = "raw-s3cr3t"
	testRestrictedName = "RESTRICTED_KEY"
)

func testSecrets() map[string]models.ComputedSecret {
	return map[string]models.ComputedSecret{
		testSecretName: {
			Name:               testSecretName,
			RawValue:           strPtr(testRawValue),
			ComputedValue:      strPtr(testComputedValue),
			RawVisibility:      "masked",
			ComputedVisibility: "masked",
			RawValueType:       models.SecretValueType{Type: "string"},
			ComputedValueType:  models.SecretValueType{Type: "string"},
			Note:               "a note",
		},
		testRestrictedName: {
			Name:               testRestrictedName,
			RawValue:           nil,
			ComputedValue:      nil,
			RawVisibility:      "restricted",
			ComputedVisibility: "restricted",
			RawValueType:       models.SecretValueType{Type: "string"},
			ComputedValueType:  models.SecretValueType{Type: "string"},
		},
	}
}

func TestSecretsTableMasking(t *testing.T) {
	tests := []struct {
		name           string
		raw            bool
		showValues     bool
		wantContain    []string
		wantNotContain []string
	}{
		{
			name:           "default table masks computed value",
			raw:            false,
			showValues:     false,
			wantContain:    []string{testSecretName, "[MASKED]", testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{testComputedValue, testRawValue},
		},
		{
			name:           "show-values table prints computed value",
			raw:            false,
			showValues:     true,
			wantContain:    []string{testSecretName, testComputedValue, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{"[MASKED]", testRawValue},
		},
		{
			name:           "raw table masks computed and raw columns by default",
			raw:            true,
			showValues:     false,
			wantContain:    []string{testSecretName, "[MASKED]", "RAW VALUE", testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{testComputedValue, testRawValue},
		},
		{
			name:           "raw table with show-values prints computed and raw values",
			raw:            true,
			showValues:     true,
			wantContain:    []string{testSecretName, testComputedValue, testRawValue, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{"[MASKED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				Secrets(testSecrets(), []string{}, false, false, tt.raw, false, false, false, tt.showValues)
			})

			for _, s := range tt.wantContain {
				if !strings.Contains(out, s) {
					t.Errorf("expected output to contain %q\noutput:\n%s", s, out)
				}
			}
			for _, s := range tt.wantNotContain {
				if strings.Contains(out, s) {
					t.Errorf("expected output not to contain %q\noutput:\n%s", s, out)
				}
			}
		})
	}
}

func TestSecretsTableMaskedCountMatchesSecrets(t *testing.T) {
	out := captureStdout(t, func() {
		Secrets(testSecrets(), []string{}, false, false, true, false, false, false, false)
	})

	// One non-restricted secret with --raw: computed column and raw column are both masked.
	if got := strings.Count(out, "[MASKED]"); got != 2 {
		t.Errorf("expected 2 [MASKED] cells, got %d\noutput:\n%s", got, out)
	}
	if got := strings.Count(out, "[RESTRICTED]"); got != 2 {
		t.Errorf("expected 2 [RESTRICTED] cells, got %d\noutput:\n%s", got, out)
	}
}

func TestSecretsJSONUnchangedByShowValues(t *testing.T) {
	for _, raw := range []bool{false, true} {
		masked := captureStdout(t, func() {
			Secrets(testSecrets(), []string{}, true, false, raw, false, false, false, false)
		})
		shown := captureStdout(t, func() {
			Secrets(testSecrets(), []string{}, true, false, raw, false, false, false, true)
		})

		if masked != shown {
			t.Errorf("raw=%v: JSON output differs between showValues=false and showValues=true\nmasked:\n%s\nshown:\n%s", raw, masked, shown)
		}
		if !strings.Contains(masked, testComputedValue) {
			t.Errorf("raw=%v: JSON output should contain the computed value\noutput:\n%s", raw, masked)
		}
		if raw && !strings.Contains(masked, testRawValue) {
			t.Errorf("JSON output with raw should contain the raw value\noutput:\n%s", masked)
		}
		if strings.Contains(masked, "[MASKED]") {
			t.Errorf("raw=%v: JSON output must never contain [MASKED]\noutput:\n%s", raw, masked)
		}
	}
}

func TestSecretsPlainUnchangedByShowValues(t *testing.T) {
	for _, showValues := range []bool{false, true} {
		out := captureStdout(t, func() {
			Secrets(testSecrets(), []string{testSecretName}, false, true, false, false, false, false, showValues)
		})
		if strings.TrimSpace(out) != testComputedValue {
			t.Errorf("showValues=%v: plain output = %q, want %q", showValues, strings.TrimSpace(out), testComputedValue)
		}

		rawOut := captureStdout(t, func() {
			Secrets(testSecrets(), []string{testSecretName}, false, true, true, false, false, false, showValues)
		})
		if strings.TrimSpace(rawOut) != testRawValue {
			t.Errorf("showValues=%v: plain raw output = %q, want %q", showValues, strings.TrimSpace(rawOut), testRawValue)
		}
	}
}
