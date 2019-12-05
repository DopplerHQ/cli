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
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

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
	return filepath.Dir(cwd)
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

// GetBool parse string into a boolean
func GetBool(value string, def bool) bool {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return b
}

// GetBoolFlag get flag parsed as a boolean
func GetBoolFlag(cmd *cobra.Command, flag string) bool {
	b, err := strconv.ParseBool(cmd.Flag(flag).Value.String())
	if err != nil {
		Err(err, "")
	}
	return b
}

// GetIntFlag get flag parsed as an int
func GetIntFlag(cmd *cobra.Command, flag string, bits int) int {
	number, err := strconv.ParseInt(cmd.Flag(flag).Value.String(), 10, bits)
	if err != nil {
		Err(err, "")
	}

	return int(number)
}

func GetDurationFlag(cmd *cobra.Command, flag string) time.Duration {
	value, err := time.ParseDuration(cmd.Flag(flag).Value.String())
	if err != nil {
		Err(err, "")
	}
	return value
}

// GetFilePath verify file path and name are provided
func GetFilePath(fullPath string, defaultPath string) string {
	if fullPath == "" {
		return defaultPath
	}

	parsedPath := filepath.Dir(fullPath)
	parsedName := filepath.Base(fullPath)

	isNameValid := (parsedName != ".") && (parsedName != "..") && (parsedName != "/") && (parsedName != string(filepath.Separator))
	if !isNameValid {
		return defaultPath
	}

	return filepath.Join(parsedPath, parsedName)
}

// ConfirmationPrompt prompt user to confirm yes/no
func ConfirmationPrompt(message string, defaultValue bool) bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
	}

	survey.AskOne(prompt, &confirm)
	return confirm
}

// CopyToClipboard copies text to the user's clipboard
func CopyToClipboard(text string) {
	if !clipboard.Unsupported {
		clipboard.WriteAll(text)
	}
}

// HostOS the host OS
func HostOS() string {
	os := runtime.GOOS

	switch os {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	}

	return os
}

// HostArch the host architecture
func HostArch() string {
	arch := runtime.GOARCH

	switch arch {
	case "amd64":
		return "64-bit"
	case "amd64p32":
	case "386":
		return "32-bit"
	}

	return arch
}
