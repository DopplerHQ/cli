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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

// DefaultFallbackDir path to the default fallback dir
var DefaultFallbackDir string

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
		enableFallback := !utils.GetBoolFlag(cmd, "no-fallback")
		fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
		fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
		exitOnWriteFailure := !utils.GetBoolFlag(cmd, "no-exit-on-write-failure")
		localConfig := configuration.LocalConfig(cmd)

		fallbackPath := ""
		if cmd.Flags().Changed("fallback") {
			fallbackPath = utils.GetFilePath(cmd.Flag("fallback").Value.String(), "")
		} else {
			fallbackPath = defaultFallbackFile(localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			if enableFallback && !utils.Exists(DefaultFallbackDir) {
				err := os.Mkdir(DefaultFallbackDir, 0700)
				if err != nil && exitOnWriteFailure {
					utils.HandleError(err, "Unable to create directory for fallback file", strings.Join(writeFailureMessage(), "\n"))
				}
			}
		}
		if fallbackPath == "" {
			utils.HandleError(errors.New("invalid fallback file path"), "")
		}

		if !enableFallback {
			flags := []string{"fallback", "fallback-only", "fallback-readonly", "no-exit-on-write-failure"}
			for _, flag := range flags {
				if cmd.Flags().Changed(flag) {
					utils.Log(fmt.Sprintf("Warning: --%s has no effect when the fallback file is disabled", flag))
				}
			}
		}

		secrets := getSecrets(cmd, localConfig, enableFallback, fallbackPath, fallbackReadonly, fallbackOnly, exitOnWriteFailure)

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

func getSecrets(cmd *cobra.Command, localConfig models.ScopedOptions, enableFallback bool, fallbackPath string, fallbackReadonly bool, fallbackOnly bool, exitOnWriteFailure bool) map[string]string {
	fetchSecrets := !(enableFallback && fallbackOnly)
	if !fetchSecrets {
		return readFallbackFile(fallbackPath)
	}

	response, httpErr := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, true)
	if httpErr != (http.Error{}) {
		if enableFallback {
			utils.LogDebug("Failed to fetch secrets from the API")
			return readFallbackFile(fallbackPath)
		}
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	// ensure the response can be parsed before proceeding
	secrets, err := parseSecrets(response)
	if err != nil {
		if enableFallback {
			utils.LogDebug("Failed to parse the API response")
			return readFallbackFile(fallbackPath)
		}
		utils.HandleError(err, "Unable to parse API response")
	}

	writeFallbackFile := enableFallback && !fallbackReadonly
	if writeFallbackFile {
		utils.LogDebug(fmt.Sprintf("Writing to fallback file %s", fallbackPath))
		err := ioutil.WriteFile(fallbackPath, response, 0600)
		if err != nil {
			utils.LogDebug("Failed to write to fallback file")
			if exitOnWriteFailure {
				utils.HandleError(err, "Unable to write fallback file", strings.Join(writeFailureMessage(), "\n"))
			} else {
				utils.LogDebug("Not exiting due to --no-exit-on-write-failure flag")
			}
		}
	}

	return secrets
}

func writeFailureMessage() []string {
	var msg []string

	msg = append(msg, "")
	msg = append(msg, color.Green.Render("Why did doppler exit?"))
	msg = append(msg, "Doppler failed to make a local backup of your secrets, known as a fallback file.")
	msg = append(msg, "The most common cause for this is insufficient permissions, including trying to use a fallback file created by a different user.")
	msg = append(msg, "")
	msg = append(msg, color.Green.Render("Why does this matter?"))
	msg = append(msg, "Without the fallback file, your secrets would be inaccessible in the event of a network outage or Doppler downtime.")
	msg = append(msg, "This could mean your development is blocked, or it could mean that your production services can't start up.")
	msg = append(msg, "")
	msg = append(msg, color.Green.Render("What should I do now?"))
	msg = append(msg, "1. You can change the location of the fallback file using the '--fallback' flag.")
	msg = append(msg, "2. You can attempt to debug and fix the local error causing the write failure.")
	msg = append(msg, "3. You can choose to ignore this error using the '--no-exit-on-write-failure' flag, but be forewarned that this is probably a really bad idea.")
	msg = append(msg, "")
	msg = append(msg, "Run 'doppler run --help' for more info.")
	msg = append(msg, "")

	return msg
}

func readFallbackFile(path string) map[string]string {
	utils.Log("Reading secrets from fallback file " + path)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			utils.HandleError(errors.New("The fallback file does not exist"))
		}

		utils.HandleError(err, "Unable to read fallback file")
	}

	response, err := ioutil.ReadFile(path)
	if err != nil {
		utils.HandleError(err, "Unable to read fallback file")
	}

	secrets, err := parseSecrets(response)
	if err != nil {
		utils.HandleError(err, "Unable to parse fallback file")
	}

	return secrets
}

func parseSecrets(response []byte) (map[string]string, error) {
	secrets := map[string]string{}
	err := json.Unmarshal(response, &secrets)
	return secrets, err
}

func defaultFallbackFile(project string, config string) string {
	fileName := fmt.Sprintf(".run-%s.json", utils.Hash(fmt.Sprintf("%s:%s", project, config)))
	return filepath.Join(DefaultFallbackDir, fileName)
}

func init() {
	DefaultFallbackDir = filepath.Join(configuration.UserConfigDir, "fallback")

	runCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	runCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	runCmd.Flags().String("fallback", "", "write secrets to this file after connecting to Doppler. secrets will be read from this file if subsequent connections are unsuccessful.")
	runCmd.Flags().Bool("no-fallback", false, "do not read or write a fallback file")
	runCmd.Flags().Bool("fallback-readonly", false, "do not create or modify the fallback file")
	runCmd.Flags().Bool("fallback-only", false, "do not request secrets from Doppler. all secrets will be read from the fallback file")
	runCmd.Flags().Bool("no-exit-on-write-failure", false, "do not exit if unable to write the fallback file")
	rootCmd.AddCommand(runCmd)
}
