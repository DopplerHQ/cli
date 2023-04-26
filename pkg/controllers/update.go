/*
Copyright © 2020 Doppler <support@doppler.com>

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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
)

// CtrlError controller errors
type CtrlError struct {
	Err     error
	Message string
}

// Error implements the native 'error' interface
func (e *CtrlError) Error() string {
	return e.Message
}

// InnerError implements the `WrappedError` interface used by http.HandleError
func (e *CtrlError) InnerError() error {
	return e.Err
}

// RunInstallScript downloads and executes the CLI install scriptm, returning true if an update was installed
func RunInstallScript() (bool, string, error) {
	startTime := time.Now()
	// download script
	script, err := http.GetCLIInstallScript()
	if err != nil {
		return false, "", &CtrlError{Err: err, Message: err.Error()}
	}
	fetchScriptDuration := time.Now().Sub(startTime).Milliseconds()

	CaptureEvent("InstallScriptDownloaded", map[string]interface{}{"durationMs": fetchScriptDuration})

	// write script to temp file
	tmpFile, err := utils.WriteTempFile("install.sh", script, 0555)
	// clean up temp file once we're done with it
	defer os.Remove(tmpFile)

	// execute script
	utils.LogDebug("Executing install script")
	command := []string{tmpFile, "--debug"}

	startTime = time.Now()
	out, err := exec.Command(command[0], command[1:]...).CombinedOutput() // #nosec G204
	executeDuration := time.Now().Sub(startTime).Milliseconds()

	strOut := string(out)
	// log output before checking error
	utils.LogDebug(fmt.Sprintf("Executing \"%s\"", strings.Join(command, " ")))
	if utils.Debug {
		// use Fprintln rather than LogDebug so that we don't display a duplicate "DEBUG" prefix
		fmt.Fprintln(os.Stderr, strOut) // nosemgrep: semgrep_configs.prohibit-print
	}
	if err != nil {
		exitCode := 1
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}

		CaptureEvent("InstallScriptFailed", map[string]interface{}{"durationMs": executeDuration, "exitCode": exitCode})

		message := "Unable to install the latest Doppler CLI"
		permissionError := exitCode == 2 || strings.Contains(strOut, "dpkg: error: requested operation requires superuser privilege")
		gnupgError := exitCode == 3
		gnupgOwnershipError := exitCode == 4
		if permissionError {
			message = "Error: update failed due to improper permissions\nPlease re-run with `sudo` or as an admin"
		} else if gnupgError {
			message = "Error: Unable to find gpg binary for signature verification\nYou can resolve this error by installing your system's gnupg package"
		} else if gnupgOwnershipError {
			message = "Error: Unable to read ~/.gnupg directory\nYou can resolve this error by running 'sudo chown -R $(whoami) ~/.gnupg'"
		}

		return false, "", &CtrlError{Err: err, Message: message}
	}

	// only capture when install is successful
	CaptureEvent("InstallScriptCompleted", map[string]interface{}{"durationMs": executeDuration})

	// find installed version within script output
	// Ex: `Installed Doppler CLI v3.7.1`
	re := regexp.MustCompile(`Installed Doppler CLI v(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(strOut)
	if matches == nil || len(matches) != 2 {
		return false, "", &CtrlError{Err: errors.New("Unable to determine new CLI version")}
	}
	// parse latest version string
	newVersion, err := version.ParseVersion(matches[1])
	if err != nil {
		return false, "", &CtrlError{Err: err, Message: "Unable to parse new CLI version"}
	}

	wasUpdated := false
	// parse current version string
	currentVersion, currVersionErr := version.ParseVersion(version.ProgramVersion)
	if currVersionErr != nil {
		// unexpected error; just consider it an update and continue executing
		wasUpdated = true
		utils.LogDebug("Unable to parse current CLI version")
		utils.LogDebugError(currVersionErr)
	}

	if !wasUpdated {
		wasUpdated = version.CompareVersions(currentVersion, newVersion) == 1
	}

	return wasUpdated, newVersion.String(), nil
}

// CLIChangeLog fetches the latest changelog
func CLIChangeLog() (map[string]models.ChangeLog, error) {
	response, err := http.GetChangelog()
	if err != nil {
		return nil, err

	}

	changes := models.ParseChangeLog(response)
	return changes, nil
}