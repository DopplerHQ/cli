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
package utils

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
)

// ProgramVersion the current version of this program
var ProgramVersion = ""

// ConfigDir get configuration directory
func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		Err(err, "Unable to determine configuration directory")
	}

	return dir
}

// HomeDir get home directory
func HomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		Err(err, "Unable to determine home directory")
	}

	return dir
}

// Exists whether path exists
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// Cwd current working directory
func Cwd() string {
	cwd, err := os.Executable()
	if err != nil {
		Err(err, "")
	}
	return path.Dir(cwd)
}

// RunCommand runs the specified command
func RunCommand(command []string, env []string) (int, error) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), err
		}

		return 1, err
	}

	return 0, nil
}

// GetBoolFlag get flag parsed as a boolean
func GetBoolFlag(cmd *cobra.Command, flag string) bool {
	value, err := strconv.ParseBool(cmd.Flag(flag).Value.String())
	if err != nil {
		Err(err, "")
	}
	return value
}

// GetIntFlag get flag parsed as an int
func GetIntFlag(cmd *cobra.Command, flag string, bits int) int {
	number, err := strconv.ParseInt(cmd.Flag(flag).Value.String(), 10, bits)
	if err != nil {
		Err(err, "")
	}

	return int(number)
}

// GetFilePath verify file path and name are provided
func GetFilePath(fullPath string, defaultPath string) string {
	if fullPath == "" {
		return defaultPath
	}

	parsedPath := filepath.Dir(fullPath)
	parsedName := filepath.Base(fullPath)

	isNameValid := (parsedName != ".") && (parsedName != "..") && (parsedName != "/")
	if !isNameValid {
		return defaultPath
	}

	return path.Join(parsedPath, parsedName)
}

// ConfirmationPrompt prompt user to confirm yes/no
func ConfirmationPrompt(message string) bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: message,
	}

	survey.AskOne(prompt, &confirm)
	return confirm
}
