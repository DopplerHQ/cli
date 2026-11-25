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
	testUnmaskedName   = "LOG_LEVEL"
	testUnmaskedCmp    = "cmp-debug"
	testUnmaskedRaw    = "raw-debug"
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
		testUnmaskedName: {
			Name:               testUnmaskedName,
			RawValue:           strPtr(testUnmaskedRaw),
			ComputedValue:      strPtr(testUnmaskedCmp),
			RawVisibility:      "unmasked",
			ComputedVisibility: "unmasked",
			RawValueType:       models.SecretValueType{Type: "string"},
			ComputedValueType:  models.SecretValueType{Type: "string"},
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
		reveal         bool
		wantContain    []string
		wantNotContain []string
	}{
		{
			name:           "default table masks only masked computed values",
			raw:            false,
			reveal:         false,
			wantContain:    []string{testSecretName, "[MASKED]", testUnmaskedCmp, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{testComputedValue, testRawValue, testUnmaskedRaw},
		},
		{
			name:           "reveal table prints computed value",
			raw:            false,
			reveal:         true,
			wantContain:    []string{testSecretName, testComputedValue, testUnmaskedCmp, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{"[MASKED]", testRawValue, testUnmaskedRaw},
		},
		{
			name:           "raw table masks only masked computed and raw values by default",
			raw:            true,
			reveal:         false,
			wantContain:    []string{testSecretName, "[MASKED]", "RAW VALUE", testUnmaskedCmp, testUnmaskedRaw, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{testComputedValue, testRawValue},
		},
		{
			name:           "raw table with reveal prints computed and raw values",
			raw:            true,
			reveal:         true,
			wantContain:    []string{testSecretName, testComputedValue, testRawValue, testUnmaskedCmp, testUnmaskedRaw, testRestrictedName, "[RESTRICTED]"},
			wantNotContain: []string{"[MASKED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := captureStdout(t, func() {
				Secrets(testSecrets(), []string{}, false, false, tt.raw, false, false, false, tt.reveal)
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

	// One masked secret with --raw: its computed and raw columns are both masked.
	// The unmasked secret's values print as-is, so it adds no [MASKED] cells.
	if got := strings.Count(out, "[MASKED]"); got != 2 {
		t.Errorf("expected 2 [MASKED] cells, got %d\noutput:\n%s", got, out)
	}
	if got := strings.Count(out, "[RESTRICTED]"); got != 2 {
		t.Errorf("expected 2 [RESTRICTED] cells, got %d\noutput:\n%s", got, out)
	}
}

func TestSecretsJSONUnchangedByReveal(t *testing.T) {
	for _, raw := range []bool{false, true} {
		masked := captureStdout(t, func() {
			Secrets(testSecrets(), []string{}, true, false, raw, false, false, false, false)
		})
		shown := captureStdout(t, func() {
			Secrets(testSecrets(), []string{}, true, false, raw, false, false, false, true)
		})

		if masked != shown {
			t.Errorf("raw=%v: JSON output differs between reveal=false and reveal=true\nmasked:\n%s\nshown:\n%s", raw, masked, shown)
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

func TestSecretsPlainUnchangedByReveal(t *testing.T) {
	for _, reveal := range []bool{false, true} {
		out := captureStdout(t, func() {
			Secrets(testSecrets(), []string{testSecretName}, false, true, false, false, false, false, reveal)
		})
		if strings.TrimSpace(out) != testComputedValue {
			t.Errorf("reveal=%v: plain output = %q, want %q", reveal, strings.TrimSpace(out), testComputedValue)
		}

		rawOut := captureStdout(t, func() {
			Secrets(testSecrets(), []string{testSecretName}, false, true, true, false, false, false, reveal)
		})
		if strings.TrimSpace(rawOut) != testRawValue {
			t.Errorf("reveal=%v: plain raw output = %q, want %q", reveal, strings.TrimSpace(rawOut), testRawValue)
		}
	}
}

func TestSecretsTableMixedRawAndComputedVisibility(t *testing.T) {
	// A secret whose raw value is masked but whose computed value is unmasked
	// (or vice versa) masks each column by its own visibility.
	secrets := map[string]models.ComputedSecret{
		testSecretName: {
			Name:               testSecretName,
			RawValue:           strPtr(testRawValue),
			ComputedValue:      strPtr(testComputedValue),
			RawVisibility:      "masked",
			ComputedVisibility: "unmasked",
		},
	}

	out := captureStdout(t, func() {
		Secrets(secrets, []string{}, false, false, true, false, false, false, false)
	})

	if !strings.Contains(out, testComputedValue) {
		t.Errorf("expected unmasked computed value %q in output:\n%s", testComputedValue, out)
	}
	if strings.Contains(out, testRawValue) {
		t.Errorf("expected masked raw value %q to be hidden:\n%s", testRawValue, out)
	}
	if got := strings.Count(out, "[MASKED]"); got != 1 {
		t.Errorf("expected 1 [MASKED] cell, got %d\noutput:\n%s", got, out)
	}
}
