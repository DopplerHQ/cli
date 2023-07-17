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
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
	"gopkg.in/gookit/color.v1"
)

const wingetPackageId = "Doppler.doppler"

// Error controller errors
type Error struct {
	Err     error
	Message string
}

// Unwrap get the original error
func (e *Error) Unwrap() error { return e.Err }

// IsNil whether the error is nil
func (e *Error) IsNil() bool { return e.Err == nil && e.Message == "" }

// CheckUpdate checks whether an update is available
func CheckUpdate(command string) (bool, models.VersionCheck) {
	// disable version checking on commands commonly used in production workflows
	// also disable when explicitly calling 'update' command to avoid checking twice
	disabledCommands := []string{"run", "secrets download", "update"}
	for _, disabledCommand := range disabledCommands {
		if command == fmt.Sprintf("doppler %s", disabledCommand) {
			utils.LogDebug("Skipping CLI upgrade check due to disallowed command")
			return false, models.VersionCheck{}
		}
	}

	if !version.PerformVersionCheck || version.IsDevelopment() {
		return false, models.VersionCheck{}
	}

	prevVersionCheck := configuration.VersionCheck()
	// don't check more often than every 24 hours
	if !time.Now().After(prevVersionCheck.CheckedAt.Add(24 * time.Hour)) {
		return false, models.VersionCheck{}
	}

	CaptureEvent("VersionCheck", nil)

	available, versionCheck, err := NewVersionAvailable(prevVersionCheck)
	if err != nil {
		return false, models.VersionCheck{}
	}

	if !available {
		utils.LogDebug("No CLI updates available")
		prevVersionCheck.CheckedAt = time.Now()
		configuration.SetVersionCheck(prevVersionCheck)
		return false, models.VersionCheck{}
	}

	if utils.IsWindows() && !utils.IsMINGW64() {
		if !installedViaWinget() {
			utils.Log(fmt.Sprintf("Update: Doppler CLI %s is available\n\nYou can update via 'scoop update doppler'\nWe recommend installing the Doppler CLI via winget for easier updates", versionCheck.LatestVersion))
			configuration.SetVersionCheck(versionCheck)
			return false, models.VersionCheck{}
		}

		if !isUpdateAvailableViaWinget(versionCheck.LatestVersion) {
			CaptureEvent("UpgradeNotAvailableViaWinget", map[string]interface{}{"version": versionCheck.LatestVersion})
			utils.LogDebug(fmt.Sprintf("Doppler CLI version %s is not yet available via winget", versionCheck.LatestVersion))
			// reuse old version so we prompt the user again
			prevVersionCheck.CheckedAt = time.Now()
			configuration.SetVersionCheck(prevVersionCheck)
			return false, models.VersionCheck{}
		}
	}

	CaptureEvent("UpgradeAvailable", nil)
	return true, versionCheck
}

func PromptToUpdate(latestVersion models.VersionCheck) {
	utils.Print(color.Green.Sprintf("An update is available."))

	changes, apiError := CLIChangeLog()
	if apiError.IsNil() {
		printer.ChangeLog(changes, 1, false)
		utils.Print("")
	}

	prompt := fmt.Sprintf("Install Doppler CLI %s", latestVersion.LatestVersion)
	if utils.ConfirmationPrompt(prompt, true) {
		CaptureEvent("UpgradeFromPrompt", nil)
		InstallUpdate(latestVersion.LatestVersion)
	} else {
		configuration.SetVersionCheck(latestVersion)
	}
}

// RunInstallScript downloads and executes the CLI install scriptm, returning true if an update was installed
func RunInstallScript() (bool, string, Error) {
	startTime := time.Now()
	// download script
	script, apiErr := http.GetCLIInstallScript()
	if !apiErr.IsNil() {
		return false, "", Error{Err: apiErr.Unwrap(), Message: apiErr.Message}
	}
	fetchScriptDuration := time.Since(startTime).Milliseconds()

	CaptureEvent("InstallScriptDownloaded", map[string]interface{}{"durationMs": fetchScriptDuration})

	// write script to temp file
	tmpFile, err := utils.WriteTempFile("install.sh", script, 0555)
	if tmpFile != "" {
		// clean up temp file once we're done with it
		defer os.Remove(tmpFile)
	}
	if err != nil {
		return false, "", Error{Err: err, Message: "Unable to save install script"}
	}

	// execute script
	utils.LogDebug("Executing install script")
	command := []string{tmpFile, "--debug"}
	utils.LogDebug(fmt.Sprintf("Executing \"%s\"", command))

	startTime = time.Now()

	var out bytes.Buffer
	var cmd *exec.Cmd
	if utils.IsWindows() {
		// must execute in sh on MINGW64 Windows to avoid "command not found" error
		c := []string{"sh"}
		c = append(c, command...)
		cmd, err = utils.RunCommand(c, os.Environ(), nil, &out, &out, true)
	} else {
		cmd, err = utils.RunCommand(command, os.Environ(), nil, &out, &out, true)
	}
	waitExitCode, waitErr := utils.WaitCommand(cmd)

	executeDuration := time.Since(startTime).Milliseconds()
	strOut := out.String()

	// log output before checking error
	if utils.Debug {
		// use Fprintln rather than LogDebug so that we don't display a duplicate "DEBUG" prefix
		fmt.Fprintln(os.Stderr, strOut) // nosemgrep: semgrep_configs.prohibit-print
	}
	if err != nil || waitErr != nil {
		if waitErr != nil {
			err = waitErr
		}
		exitCode := 1
		if waitExitCode != 0 {
			exitCode = waitExitCode
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

		return false, "", Error{Err: err, Message: message}
	}

	// only capture when install is successful
	CaptureEvent("InstallScriptCompleted", map[string]interface{}{"durationMs": executeDuration})

	// find installed version within script output
	// Ex: `Installed Doppler CLI v3.7.1`
	re := regexp.MustCompile(`Installed Doppler CLI v(\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(strOut)
	if matches == nil || len(matches) != 2 {
		return false, "", Error{Err: errors.New("Unable to determine new CLI version")}
	}
	// parse latest version string
	newVersion, err := version.ParseVersion(matches[1])
	if err != nil {
		return false, "", Error{Err: err, Message: "Unable to parse new CLI version"}
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

	return wasUpdated, newVersion.String(), Error{}
}

// CLIChangeLog fetches the latest changelog
func CLIChangeLog() (map[string]models.ChangeLog, http.Error) {
	response, apiError := http.GetChangelog()
	if !apiError.IsNil() {
		return nil, apiError

	}

	changes := models.ParseChangeLog(response)
	return changes, http.Error{}
}

func InstallUpdate(version string) {
	utils.Print("Updating...")

	var wasUpdated bool
	var installedVersion string
	var controllerErr Error
	if utils.IsWindows() && !utils.IsMINGW64() {
		if installedViaWinget() {
			if err := updateViaWinget(version); err != nil {
				utils.HandleError(err, "Unable to execute winget")
			}

			utils.LogDebug("Executing winget in background, CLI is now exiting")
			os.Exit(0)
		} else {
			utils.HandleError(fmt.Errorf("updates are not supported when installed via scoop. Please install the Doppler CLI via winget or update manually via `scoop update doppler`"))
		}
	} else {
		wasUpdated, installedVersion, controllerErr = RunInstallScript()
	}
	if !controllerErr.IsNil() {
		utils.HandleError(controllerErr.Unwrap(), controllerErr.Message)
	}

	if wasUpdated {
		utils.Print(fmt.Sprintf("Installed CLI %s", installedVersion))

		if changes, apiError := CLIChangeLog(); apiError.IsNil() {
			utils.Print("\nWhat's new:")
			printer.ChangeLog(changes, 1, false)
			utils.Print("\nTip: run 'doppler changelog' to see all latest changes")
		}

		utils.Print("")
	} else {
		utils.Print("You are already running the latest version")
	}

	versionCheck := models.VersionCheck{LatestVersion: installedVersion, CheckedAt: time.Now()}
	configuration.SetVersionCheck(versionCheck)
}

func installedViaWinget() bool {
	utils.LogDebug("Checking if CLI is installed via winget")
	command := fmt.Sprintf("winget list --id %s -n 1 --exact --disable-interactivity", wingetPackageId)
	utils.LogDebug(fmt.Sprintf("Executing \"%s\"", command))

	var out io.Writer = nil
	if utils.CanLogDebug() {
		out = os.Stderr
	}
	cmd, err := utils.RunCommandString(command, os.Environ(), nil, out, out, true)
	if err != nil {
		utils.LogDebugError(err)
		return false
	}
	_, err = utils.WaitCommand(cmd)
	if err != nil {
		utils.LogDebugError(err)
	}
	return err == nil
}

func isUpdateAvailableViaWinget(updateVersion string) bool {
	utils.LogDebug("Checking if CLI update is available via winget")
	command := fmt.Sprintf("winget list --id %s -n 1 --exact --disable-interactivity", wingetPackageId)
	utils.LogDebug(fmt.Sprintf("Executing \"%s\"", command))

	var out bytes.Buffer
	cmd, err := utils.RunCommandString(command, os.Environ(), nil, &out, &out, true)
	if err != nil {
		utils.LogDebugError(err)
		return false
	}

	_, err = utils.WaitCommand(cmd)
	if err != nil {
		utils.LogDebugError(err)
		// not installed via winget
		return false
	}
	strOut := out.String()
	utils.LogDebug(strOut)

	// Ex: `Doppler.doppler 3.63.1 3.64.0 winget`
	re := regexp.MustCompile(fmt.Sprintf(`%s\s+\d+\.\d+\.\d+\s+%s\s+winget`, wingetPackageId, strings.TrimPrefix(updateVersion, "v")))

	matches := re.FindStringSubmatch(strOut)
	return len(matches) > 0
}

func updateViaWinget(version string) error {
	CaptureEvent("WingetUpgradeInitiated", nil)

	command := fmt.Sprintf("winget upgrade --id %s --exact --disable-interactivity --version %s", wingetPackageId, strings.TrimPrefix(version, "v"))

	utils.LogDebug(fmt.Sprintf("Executing \"%s\"", command))
	_, err := utils.RunCommandString(command, os.Environ(), nil, os.Stdout, os.Stderr, true)
	if err != nil {
		CaptureEvent("WingetUpgradeFailed", nil)
		return err
	}

	return nil
}
