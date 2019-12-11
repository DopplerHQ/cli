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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command with secrets injected into the environment",
	Long: `Run a command with secrets injected into the environment

To view the CLI's active configuration, run ` + "`doppler configure debug`",
	Example: `doppler run printenv
doppler run -- printenv
doppler run --token=123 -- printenv`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
		fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
		fallbackPath := utils.GetFilePath(cmd.Flag("fallback").Value.String(), "")

		if cmd.Flags().Changed("fallback") && fallbackPath == "" {
			utils.HandleError(errors.New("invalid fallback file path"), "")
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

func getSecrets(cmd *cobra.Command, localConfig models.ScopedOptions, fallbackPath string, fallbackReadonly bool, fallbackOnly bool) map[string]string {
	useFallbackFile := (fallbackPath != "")
	if useFallbackFile && fallbackOnly {
		return readFallbackFile(fallbackPath)
	}

	response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
	if err != (http.Error{}) {
		if useFallbackFile {
			return readFallbackFile(fallbackPath)
		}
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if useFallbackFile && !fallbackReadonly {
		err := ioutil.WriteFile(fallbackPath, response, 0600)
		if err != nil {
			utils.HandleError(err, "Unable to write fallback file")
		}
	}

	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		utils.HandleError(parseErr, "Unable to parse API response")
	}

	secretsStrings := map[string]string{}
	for key, value := range secrets {
		secretsStrings[key] = value.ComputedValue
	}

	return secretsStrings
}

func readFallbackFile(path string) map[string]string {
	utils.Log("Using fallback file")
	response, err := ioutil.ReadFile(path)
	if err != nil {
		utils.HandleError(err, "Unable to read fallback file")
	}

	secrets, err := models.ParseSecrets(response)
	if err != nil {
		utils.HandleError(err, "Unable to parse fallback file")
	}

	secretsStrings := map[string]string{}
	for key, value := range secrets {
		secretsStrings[key] = value.ComputedValue
	}

	return secretsStrings
}

func init() {
	runCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	runCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")

	runCmd.Flags().String("fallback", "", "write secrets to this file after connecting to Doppler. secrets will be read from this file if future connection attempts are unsuccessful.")
	runCmd.Flags().Bool("fallback-readonly", false, "do not update or modify the fallback file")
	runCmd.Flags().Bool("fallback-only", false, "do not request secrets from Doppler. all secrets will be read directly from the fallback file")

	rootCmd.AddCommand(runCmd)
}
