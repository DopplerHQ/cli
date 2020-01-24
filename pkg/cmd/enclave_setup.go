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
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup the Doppler CLI for Enclave",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		silent := utils.GetBoolFlag(cmd, "silent")
		promptUser := !utils.GetBoolFlag(cmd, "no-prompt")
		scope := cmd.Flag("scope").Value.String()
		localConfig := configuration.LocalConfig(cmd)
		scopedConfig := configuration.Get(scope)

		flagsFromEnvironment := []string{}

		project := ""
		switch localConfig.EnclaveProject.Source {
		case models.FlagSource.String():
			project = localConfig.EnclaveProject.Value
		case models.EnvironmentSource.String():
			flagsFromEnvironment = append(flagsFromEnvironment, "ENCLAVE_PROJECT")
			project = localConfig.EnclaveProject.Value
		default:
			if !promptUser {
				utils.HandleError(errors.New("project must be specified via --project flag or ENCLAVE_PROJECT environment variable when using --no-prompt"))
			}

			projects, httpErr := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
			if !httpErr.IsNil() {
				utils.HandleError(httpErr.Unwrap(), httpErr.Message)
			}
			if len(projects) == 0 {
				utils.HandleError(errors.New("you do not have access to any projects"))
			}

			var projectOptions []string
			defaultOption := ""
			for _, val := range projects {
				option := val.Name + " (" + val.ID + ")"
				projectOptions = append(projectOptions, option)

				// reselect previously-configured value
				if scopedConfig.EnclaveProject.Value == val.ID {
					defaultOption = option
				}
			}

			prompt := &survey.Select{
				Message: "Select a project:",
				Options: projectOptions,
				Default: defaultOption,
			}
			err := survey.AskOne(prompt, &project)
			if err != nil {
				utils.HandleError(err)
			}

			for _, val := range projects {
				if strings.HasSuffix(project, "("+val.ID+")") {
					project = val.ID
					break
				}
			}
		}

		config := ""
		switch localConfig.EnclaveConfig.Source {
		case models.FlagSource.String():
			config = localConfig.EnclaveConfig.Value
		case models.EnvironmentSource.String():
			flagsFromEnvironment = append(flagsFromEnvironment, "ENCLAVE_CONFIG")
			config = localConfig.EnclaveConfig.Value
		default:
			if !promptUser {
				utils.HandleError(errors.New("config must be specified via --config flag or ENCLAVE_CONFIG environment variable when using --no-prompt"))
			}

			configs, apiError := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
			if !apiError.IsNil() {
				utils.HandleError(apiError.Unwrap(), apiError.Message)
			}
			if len(configs) == 0 {
				utils.HandleError(errors.New("your project does not have any configs"))
			}

			var configOptions []string
			for _, val := range configs {
				configOptions = append(configOptions, val.Name)
			}

			prompt := &survey.Select{
				Message: "Select a config:",
				Options: configOptions,
				Default: scopedConfig.EnclaveConfig.Value,
			}
			err := survey.AskOne(prompt, &config)
			if err != nil {
				utils.HandleError(err)
			}
		}

		configuration.Set(scope, map[string]string{
			models.ConfigEnclaveProject.String(): project,
			models.ConfigEnclaveConfig.String():  config,
		})

		if !silent {
			if len(flagsFromEnvironment) > 0 {
				fmt.Println("Using " + strings.Join(flagsFromEnvironment, " and ") + " from the environment. To disable this, use --no-read-env.")
			}

			// do not fetch the LocalConfig since we do not care about env variables or cmd flags
			conf := configuration.Get(scope)
			valuesToPrint := []string{models.ConfigEnclaveConfig.String(), models.ConfigEnclaveProject.String()}
			printer.ScopedConfigValues(conf, valuesToPrint, models.ScopedPairs(&conf), utils.OutputJSON, false)
		}
	},
}

func init() {
	setupCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	setupCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	setupCmd.Flags().Bool("silent", false, "disable text output")
	setupCmd.Flags().Bool("no-prompt", false, "do not prompt for information. if the project or config is not specified, an error will be thrown.")
	enclaveCmd.AddCommand(setupCmd)
}
