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
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:       "completion",
	Short:     "Print shell completion script",
	ValidArgs: []string{"bash", "zsh", "fish"},
	Args:      cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := getShell(args)

		if strings.HasSuffix(shell, "/bash") {
			if err := cmd.Root().GenBashCompletion(os.Stdout); err != nil {
				utils.HandleError(err, "Unable to generate bash completions.")
			}
		} else if strings.HasSuffix(shell, "/zsh") {
			if err := cmd.Root().GenZshCompletion(os.Stdout); err != nil {
				utils.HandleError(err, "Unable to generate zsh completions.")
			}
		} else if strings.HasSuffix(shell, "/fish") {
			if err := cmd.Root().GenFishCompletion(os.Stdout, true); err != nil {
				utils.HandleError(err, "Unable to generate fish completions.")
			}
		} else {
			utils.HandleError(errors.New("Your shell is not supported"))
		}
	},
}

var completionInstallCmd = &cobra.Command{
	Use:       "install [shell]",
	Short:     "Install completions for the current shell",
	ValidArgs: []string{"bash"},
	Args:      cobra.MaximumNArgs(1),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// This function is used to prevent calling the global PersistentPreRun handler, which is responsible for creating the config directory.
		// This command is called by our install.sh script, which is often run with `sudo`. If the effective user differs from the real user,
		// the config directory will be created with the wrong ownership, resulting in an error on all subsequent CLI runs. Thus we opt to skip
		// creation entirely since this command doesn't rely on any config settings.

		// ensure we still process flags like debug, silent, etc.
		loadFlags(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		shell := getShell(args)

		var buf bytes.Buffer
		var path string
		var name string
		// Track if using fallback path (requires manual fpath configuration)
		var usingFallbackPath bool

		if utils.IsWindows() {
			utils.HandleError(errors.New("Completion files are not supported on Windows. You can use completion files with Windows Subsystem for Linux (WSL)."))
		}

		if strings.HasSuffix(shell, "/bash") {
			if err := cmd.Root().GenBashCompletion(&buf); err != nil {
				utils.HandleError(err, "Unable to generate bash completions.")
			}
			name = "doppler"
			if utils.IsMacOS() {
				path = "/usr/local/etc/bash_completion.d"
			} else {
				path = "/etc/bash_completion.d"
			}
		} else if strings.HasSuffix(shell, "/zsh") {
			if err := cmd.Root().GenZshCompletion(&buf); err != nil {
				utils.HandleError(err, "Unable to generate zsh completions.")
			}
			// zsh completions file start with an underscore
			name = "_doppler"
			// Try standard path first, fall back to user directory if not writable
			standardPath := "/usr/local/share/zsh/site-functions"
			if isDirectoryWritable(standardPath) {
				path = standardPath
			} else {
				// Fall back to XDG_DATA_HOME/doppler or ~/.local/share/doppler
				dataHome := os.Getenv("XDG_DATA_HOME")
				if dataHome == "" {
					homeDir, err := os.UserHomeDir()
					if err != nil {
						utils.HandleError(err, "Unable to determine home directory")
					}
					dataHome = fmt.Sprintf("%s/.local/share", homeDir)
				}
				path = fmt.Sprintf("%s/doppler/zsh/completions", dataHome)
				usingFallbackPath = true
			}
		} else {
			utils.HandleError(errors.New("Your shell is not supported"))
		}

		// create directory and intermediate directories if they don't exist
		if !utils.Exists(path) {
			// using 755 to mimic expected /etc/ perms
			err := os.MkdirAll(path, 0755) // #nosec G301
			if err != nil {
				utils.HandleError(err, "Unable to write completion file")
			}
		}

		filePath := fmt.Sprintf("%s/%s", path, name)
		utils.LogDebug(fmt.Sprintf("Writing completion file to %s", filePath))
		if err := utils.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			utils.HandleError(err, "Unable to write completion file")
		}

		utils.Print("Completions have been installed. Restart your shell to apply.")
		utils.Print("")
		if strings.HasSuffix(shell, "/zsh") && usingFallbackPath {
			utils.Print("To enable completions, add the following to your ~/.zshrc:")
			utils.Print("")
			utils.Print(fmt.Sprintf("  fpath=(%s $fpath)", path))
		} else if strings.HasSuffix(shell, "/bash") {
			if utils.IsMacOS() {
				utils.Print("Note: The homebrew 'bash-completion' package is required for completions to work. See https://docs.brew.sh/Shell-Completion for more info.")
			} else {
				utils.Print("Note: The 'bash-completion' package is required for completions to work. See https://github.com/scop/bash-completion for more info.")
			}
		}
	},
}

func getShell(args []string) string {
	shell := os.Getenv("SHELL")
	if len(args) > 0 {
		shell = fmt.Sprintf("%s", args[0])
	}
	if shell == "" {
		utils.HandleError(errors.New("Unable to determine current shell"), "Please provide your shell name as an argument")
	}

	// normalize shell
	if !strings.HasPrefix(shell, "/") {
		shell = fmt.Sprintf("/%s", shell)
	}

	return shell
}

func isDirectoryWritable(path string) bool {
	// If directory exists, check if it's writable
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return false
		}
		// Try to create a temp file to test write access
		testFile := fmt.Sprintf("%s/.doppler_write_test", path)
		if f, err := os.Create(testFile); err == nil {
			f.Close()
			os.Remove(testFile)
			return true
		}
		return false
	}

	// Directory doesn't exist, check if parent is writable
	parent := filepath.Dir(path)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		testFile := fmt.Sprintf("%s/.doppler_write_test", parent)
		if f, err := os.Create(testFile); err == nil {
			f.Close()
			os.Remove(testFile)
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(completionInstallCmd)
}
