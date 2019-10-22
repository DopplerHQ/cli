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
	api "doppler-cli/api"
	configuration "doppler-cli/config"
	dopplerErrors "doppler-cli/errors"
	"doppler-cli/utils"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var deployHost string
var key string
var project string
var config string

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "A brief description of your command",
	Long: `Run a command with secrets injected into the environment

Usage:
doppler run printenv
doppler run -- printenv
doppler run --key=123 -- printenv`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		silent := utils.GetBoolFlag(cmd, "silent")

		localConfig := configuration.LocalConfig(cmd)
		_, secrets := api.GetDeploySecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, true)

		env := os.Environ()
		excludedKeys := []string{"PATH", "PS1", "HOME"}
		for name, value := range secrets {
			addKey := true
			for _, excludedKey := range excludedKeys {
				if excludedKey == name {
					addKey = false
					break
				}
			}

			if addKey {
				env = append(env, fmt.Sprintf("%s=%s", name, value))
			}
		}

		err := utils.RunCommand(args, env, !silent)
		if err != nil {
			fmt.Println(fmt.Sprintf("Error trying to execute command: %s", args))
			utils.Err(err)
		}
	},
}

func init() {
	runCmd.Flags().Bool("silent", false, "don't output the response")
	runCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	runCmd.Flags().String("config", "", "doppler config (e.g. dev)")

	rootCmd.AddCommand(runCmd)
}
