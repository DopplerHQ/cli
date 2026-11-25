package printer

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

const (
	testAuthToken = "dp.st.dev.abcdefghijklmnopqrstuvwxyz0123456789"
	testConfig    = "dev_personal"
)

func testScopedOptions() models.ScopedOptions {
	return models.ScopedOptions{
		Token:         models.ScopedOption{Value: testAuthToken, Scope: "/", Source: models.ConfigFileSource.String()},
		EnclaveConfig: models.ScopedOption{Value: testConfig, Scope: "/", Source: models.ConfigFileSource.String()},
	}
}

// tableColumn joins the trimmed cells of column col (0-based) on each line of
// Table output, so that a value that Table wraps reads as one string.
func tableColumn(out string, col int) string {
	var b strings.Builder
	for _, line := range strings.Split(out, "\n") {
		if cells := strings.Split(line, "│"); len(cells) > col+1 {
			b.WriteString(strings.TrimSpace(cells[col+1]))
		}
	}
	return b.String()
}

func TestScopedConfigValuesTableRedactsToken(t *testing.T) {
	conf := testScopedOptions()
	args := []string{models.ConfigToken.String(), models.ConfigEnclaveConfig.String()}

	out := captureStdout(t, func() {
		ScopedConfigValues(conf, args, models.ScopedOptionsMap(&conf), false, false, false)
	})

	// Table wraps long cells over several lines, so the full token never
	// appears on one line of out. Join the value column first.
	if values := tableColumn(out, 1); strings.Contains(values, testAuthToken) {
		t.Errorf("table output contains the full auth token:\n%s", out)
	}
	if redacted := utils.RedactAuthToken(testAuthToken); !strings.Contains(out, redacted) {
		t.Errorf("table output does not contain redacted token %q:\n%s", redacted, out)
	}
	if !strings.Contains(out, testConfig) {
		t.Errorf("table output does not contain config value %q:\n%s", testConfig, out)
	}
}

func TestScopedConfigValuesJSONShowsToken(t *testing.T) {
	conf := testScopedOptions()
	args := []string{models.ConfigToken.String(), models.ConfigEnclaveConfig.String()}

	out := captureStdout(t, func() {
		ScopedConfigValues(conf, args, models.ScopedOptionsMap(&conf), true, false, false)
	})

	var got map[string]string
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("unable to parse JSON output: %v\n%s", err, out)
	}
	if got[models.ConfigToken.String()] != testAuthToken {
		t.Errorf("JSON token = %q, want %q", got[models.ConfigToken.String()], testAuthToken)
	}
	if got[models.ConfigEnclaveConfig.String()] != testConfig {
		t.Errorf("JSON config = %q, want %q", got[models.ConfigEnclaveConfig.String()], testConfig)
	}
}

func TestScopedConfigValuesPlainShowsToken(t *testing.T) {
	conf := testScopedOptions()
	args := []string{models.ConfigToken.String(), models.ConfigEnclaveConfig.String()}

	out := captureStdout(t, func() {
		ScopedConfigValues(conf, args, models.ScopedOptionsMap(&conf), false, true, false)
	})

	want := testAuthToken + "\n" + testConfig + "\n"
	if out != want {
		t.Errorf("plain output = %q, want %q", out, want)
	}
}
