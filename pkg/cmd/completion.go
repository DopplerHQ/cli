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
	"fmt"
	"os"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:       "completion",
	Short:     "Print shell completion script",
	ValidArgs: []string{"bash", "zsh"},
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
		} else {
			utils.HandleError(fmt.Errorf("Your shell is not supported"))
		}
	},
}

var completionInstallCmd = &cobra.Command{
	Use:       "install [shell]",
	Short:     "Install completions for the current shell",
	ValidArgs: []string{"bash"},
	Args:      cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		shell := getShell(args)

		var buf bytes.Buffer
		var path string
		var name string
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
		} else {
			utils.HandleError(fmt.Errorf("Your shell is not supported"))
		}

		// create directory if it doesn't exist
		if !utils.Exists(path) {
			// using 755 to mimic expected /etc/ perms
			err := os.Mkdir(path, 0755) // #nosec G301
			if err != nil {
				utils.HandleError(err, "Unable to write completion file")
			}
		}

		filePath := fmt.Sprintf("%s/%s", path, name)
		utils.LogDebug(fmt.Sprintf("Writing completion file to %s", filePath))
		if err := utils.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			utils.HandleError(err, "Unable to write completion file")
		}

		utils.Log("Your shell has been configured for Doppler CLI completions! Restart your shell to apply.")
		utils.Log("")
		if utils.IsMacOS() {
			utils.Log("Note: The homebrew 'bash-completion' package is required for completions to work. See https://docs.brew.sh/Shell-Completion for more info.")
		} else {
			utils.Log("Note: The 'bash-completion' package is required for completions to work. See https://github.com/scop/bash-completion for more info.")
		}
	},
}

func getShell(args []string) string {
	shell := os.Getenv("SHELL")
	if len(args) > 0 {
		shell = fmt.Sprintf("%s", args[0])
	}
	if shell == "" {
		utils.HandleError(fmt.Errorf("Unable to determine current shell"), "Please provide your shell name as an argument")
	}

	// normalize shell
	if !strings.HasPrefix(shell, "/") {
		shell = fmt.Sprintf("/%s", shell)
	}

	return shell
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(completionInstallCmd)
}
