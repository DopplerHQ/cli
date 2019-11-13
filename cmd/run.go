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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/DopplerHQ/cli/api"
	"github.com/DopplerHQ/cli/configuration"
	"github.com/DopplerHQ/cli/models"
	"github.com/DopplerHQ/cli/utils"
	"github.com/spf13/cobra"
)

var deployHost string
var key string
var project string
var config string

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command with secrets injected into the environment",
	Long: `Run a command with secrets injected into the environment

Usage:
doppler run printenv
doppler run -- printenv
doppler run --key=123 -- printenv`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
		fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
		fallbackPath := utils.GetFilePath(cmd.Flag("fallback").Value.String(), "")

		if cmd.Flags().Changed("fallback") && fallbackPath == "" {
			utils.Err(errors.New("invalid fallback file path"), "")
		}

		localConfig := configuration.LocalConfig(cmd)
		secrets := getSecrets(cmd, localConfig, fallbackPath, fallbackReadonly, fallbackOnly)

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

		exitCode, err := utils.RunCommand(args, env)
		if err != nil || exitCode != 0 {
			utils.ErrExit(err, exitCode, fmt.Sprintf("Error trying to execute command: %s", strings.Join(args, " ")))
		}
	},
}

func getSecrets(cmd *cobra.Command, localConfig models.ScopedConfig, fallbackPath string, fallbackReadonly bool, fallbackOnly bool) map[string]string {
	useFallbackFile := (fallbackPath != "")
	if useFallbackFile && fallbackOnly {
		return readFallbackFile(fallbackPath)
	}

	response, err := api.GetDeploySecrets(cmd, localConfig.DeployHost.Value, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value)

	if !useFallbackFile && err != nil {
		utils.Err(err)
	}

	if useFallbackFile {
		if err != nil {
			return readFallbackFile(fallbackPath)
		}

		if !fallbackReadonly {
			err := ioutil.WriteFile(fallbackPath, response, 0600)
			if err != nil {
				utils.Err(err, "Unable to write fallback file")
			}
		}
	}

	secrets, err := api.ParseDeploySecrets(response)
	if err != nil {
		utils.Err(err, "Unable to parse response")
	}

	return secrets
}

func readFallbackFile(path string) map[string]string {
	utils.Log("Using fallback file")
	response, err := ioutil.ReadFile(path)
	if err != nil {
		utils.Err(err, "Unable to read fallback file")
	}

	secrets, err := api.ParseDeploySecrets(response)
	if err != nil {
		utils.Err(err, "Unable to parse fallback file")
	}

	return secrets
}

func init() {
	runCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	runCmd.Flags().String("config", "", "doppler config (e.g. dev)")

	runCmd.Flags().String("fallback", "", "write secrets to this file after connecting to Doppler. secrets will be read from this file if future connection attempts are unsuccessful.")
	runCmd.Flags().Bool("fallback-readonly", false, "don't update or modify the fallback file")
	runCmd.Flags().Bool("fallback-only", false, "don't request secrets from Doppler. all secrets will be read directly from the fallback file")

	rootCmd.AddCommand(runCmd)
}
