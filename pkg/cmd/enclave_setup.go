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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup the Doppler CLI for Enclave",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		silent := utils.GetBoolFlag(cmd, "silent")
		promptUser := !utils.GetBoolFlag(cmd, "no-prompt")
		canSaveToken := !utils.GetBoolFlag(cmd, "no-save-token")
		scope := cmd.Flag("scope").Value.String()
		localConfig := configuration.LocalConfig(cmd)
		scopedConfig := configuration.Get(scope)

		utils.RequireValue("token", localConfig.Token.Value)

		saveToken := false
		if canSaveToken {
			// save the token when it's passed via command line
			switch localConfig.Token.Source {
			case models.FlagSource.String():
				saveToken = true
			case models.EnvironmentSource.String():
				utils.Log(valueFromEnvironmentNotice("DOPPLER_TOKEN"))
				saveToken = true
			}
		}

		currentProject := localConfig.EnclaveProject.Value
		selectedProject := ""

		switch localConfig.EnclaveProject.Source {
		case models.FlagSource.String():
			selectedProject = localConfig.EnclaveProject.Value
		case models.EnvironmentSource.String():
			utils.Log(valueFromEnvironmentNotice("ENCLAVE_PROJECT"))
			selectedProject = localConfig.EnclaveProject.Value
		default:
			projects, httpErr := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
			if !httpErr.IsNil() {
				utils.HandleError(httpErr.Unwrap(), httpErr.Message)
			}
			if len(projects) == 0 {
				utils.HandleError(errors.New("you do not have access to any projects"))
			}

			selectedProject = selectProject(projects, scopedConfig.EnclaveProject.Value, promptUser)
			if selectedProject == "" {
				utils.HandleError(errors.New("Invalid project"))
			}
		}

		selectedConfiguredProject := selectedProject == currentProject
		selectedConfig := ""

		switch localConfig.EnclaveConfig.Source {
		case models.FlagSource.String():
			selectedConfig = localConfig.EnclaveConfig.Value
		case models.EnvironmentSource.String():
			utils.Log(valueFromEnvironmentNotice("ENCLAVE_CONFIG"))
			selectedConfig = localConfig.EnclaveConfig.Value
		default:
			configs, apiError := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, selectedProject)
			if !apiError.IsNil() {
				utils.HandleError(apiError.Unwrap(), apiError.Message)
			}
			if len(configs) == 0 {
				utils.Log("You project does not have any configs")
				break
			}

			selectedConfig = selectConfig(configs, selectedConfiguredProject, scopedConfig.EnclaveConfig.Value, promptUser)
			if selectedConfig == "" {
				utils.HandleError(errors.New("Invalid config"))
			}
		}

		configToSave := map[string]string{
			models.ConfigEnclaveProject.String(): selectedProject,
			models.ConfigEnclaveConfig.String():  selectedConfig,
		}
		if saveToken {
			configToSave[models.ConfigToken.String()] = localConfig.Token.Value
		}
		configuration.Set(scope, configToSave)

		if !silent {
			// do not fetch the LocalConfig since we do not care about env variables or cmd flags
			conf := configuration.Get(scope)
			valuesToPrint := []string{models.ConfigEnclaveConfig.String(), models.ConfigEnclaveProject.String()}
			if saveToken {
				valuesToPrint = append(valuesToPrint, models.ConfigToken.String())
			}
			printer.ScopedConfigValues(conf, valuesToPrint, models.ScopedPairs(&conf), utils.OutputJSON, false, false)
		}
	},
}

func selectProject(projects []models.ProjectInfo, prevConfiguredProject string, promptUser bool) string {
	var options []string
	var defaultOption string
	for _, val := range projects {
		option := val.Name
		if val.Name != val.ID {
			option = fmt.Sprintf("%s (%s)", option, val.ID)
		}
		options = append(options, option)

		if val.ID == prevConfiguredProject {
			defaultOption = option
		}
	}

	if len(projects) == 1 {
		// the user is expecting to a prompt, so print a message instead
		if promptUser {
			utils.Log(fmt.Sprintf("%s %s", color.Bold.Render("Selected only available project:"), options[0]))
		}
		return projects[0].ID
	}

	if !promptUser {
		utils.HandleError(errors.New("project must be specified via --project flag or ENCLAVE_PROJECT environment variable when using --no-prompt"))
	}

	selectedProject := utils.SelectPrompt("Select a project:", options, defaultOption)
	for _, val := range projects {
		if selectedProject == val.ID || strings.HasSuffix(selectedProject, "("+val.ID+")") {
			return val.ID
		}
	}

	return ""
}

func selectConfig(configs []models.ConfigInfo, selectedConfiguredProject bool, prevConfiguredConfig string, promptUser bool) string {
	var options []string
	var defaultOption string
	for _, val := range configs {
		option := val.Name
		options = append(options, option)

		// make previously selected config the default when re-using the previously selected project
		if selectedConfiguredProject && val.Name == prevConfiguredConfig {
			defaultOption = val.Name
		}
	}

	if len(configs) == 1 {
		config := configs[0].Name
		// the user is expecting to a prompt, so print a message instead
		if promptUser {
			utils.Log(fmt.Sprintf("%s %s", color.Bold.Render("Selected only available config:"), config))
		}
		return config
	}

	if !promptUser {
		utils.HandleError(errors.New("config must be specified via --config flag or ENCLAVE_CONFIG environment variable when using --no-prompt"))
	}

	selectedConfig := utils.SelectPrompt("Select a config:", options, defaultOption)
	return selectedConfig
}

func valueFromEnvironmentNotice(name string) string {
	return fmt.Sprintf("Using %s from the environment. To disable this, use --no-read-env.", name)
}

func init() {
	setupCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	setupCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	setupCmd.Flags().Bool("no-prompt", false, "do not prompt for information. if the project or config is not specified, an error will be thrown.")
	setupCmd.Flags().Bool("no-save-token", false, "do not save the token to the config when passed via flag or environment variable.")
	enclaveCmd.AddCommand(setupCmd)
}
