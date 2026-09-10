package configuration

import (
	"path/filepath"
	"testing"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
)

// The on-disk config location must derive from the injectable branding vars, so a
// renamed build (e.g. doppler-agent) reads/writes ~/.doppler-agent and can't touch a
// production doppler install's credentials.
func TestConfigPathsFollowBranding(t *testing.T) {
	if configFileName != version.ConfigFileName {
		t.Fatalf("configFileName = %q, want it wired to version.ConfigFileName %q", configFileName, version.ConfigFileName)
	}
	wantDir := filepath.Join(utils.HomeDir(), version.ConfigDirName)
	if UserConfigDir != wantDir {
		t.Fatalf("default UserConfigDir = %q, want %q (from version.ConfigDirName)", UserConfigDir, wantDir)
	}
	if UserConfigFile != filepath.Join(wantDir, version.ConfigFileName) {
		t.Fatalf("UserConfigFile = %q, want it under the branded dir/file", UserConfigFile)
	}
}
