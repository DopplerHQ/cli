//go:build !windows
// +build !windows

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
package controllers

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// set when tests run with the race detector (see race_enabled_test.go)
var raceEnabled bool

// The mount's writer goroutine and its cleanup func both use the cleanup flag. This orders them so the writer reads
// the flag after the cleanup has set it: a reader holds the pipe open so the writer keeps cycling, the cleanup deletes
// the pipe, and closing the reader makes the writer's next open fail because the pipe no longer exists.
func TestMountSecretsCleanupWhileWriterIsActive(t *testing.T) {
	mountPath := filepath.Join(t.TempDir(), "secrets.json")
	_, cleanup, err := MountSecrets([]byte(`{"FOO":"bar"}`), mountPath, 0)
	if !err.IsNil() {
		t.Fatal(err.Unwrap())
	}

	// blocks until the writer opens the pipe
	reader, openErr := os.OpenFile(mountPath, os.O_RDONLY, os.ModeNamedPipe) // #nosec G304
	if openErr != nil {
		t.Fatal(openErr)
	}

	cleanup()
	assert.NoFileExists(t, mountPath)

	reader.Close()
	// give the writer time to notice; it must stop quietly rather than exiting the process with an error
	time.Sleep(500 * time.Millisecond)
	assert.NoFileExists(t, mountPath)
}

// CI doesn't run tests with the race detector, so run the test above under it
func TestMountSecretsCleanupHasNoDataRaces(t *testing.T) {
	if raceEnabled {
		t.Skip("the race detector is already enabled")
	}
	if testing.Short() {
		t.Skip("building with the race detector is slow")
	}

	out, err := exec.Command("go", "test", "-race", "-count=1", "-run", "^TestMountSecretsCleanupWhileWriterIsActive$", ".").CombinedOutput() // #nosec G204
	if strings.Contains(string(out), "WARNING: DATA RACE") {
		t.Fatalf("the race detector reported a data race:\n%s", out)
	}
	if err != nil {
		t.Fatalf("unable to run test with the race detector: %v\n%s", err, out)
	}
}
