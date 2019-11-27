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
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/DopplerHQ/cli/pkg/api"
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup the Doppler cli",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		silent := utils.GetBoolFlag(cmd, "silent")
		scope := cmd.Flag("scope").Value.String()
		localConfig := configuration.LocalConfig(cmd)
		projects, err := api.GetProjects(cmd, localConfig.APIHost.Value, localConfig.Token.Value)
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}

		project := ""
		if cmd.Flags().Changed("project") {
			project = localConfig.Project.Value
		} else {
			var projectOptions []string
			for _, val := range projects {
				projectOptions = append(projectOptions, val.Name+" ("+val.ID+")")
			}
			prompt := &survey.Select{
				Message: "Select a project:",
				Options: projectOptions,
			}
			err := survey.AskOne(prompt, &project)
			if err != nil {
				utils.Err(err)
			}

			for _, val := range projects {
				if strings.HasSuffix(project, "("+val.ID+")") {
					project = val.ID
					break
				}
			}
		}

		config := ""
		if cmd.Flags().Changed("config") {
			config = localConfig.Config.Value
		} else {
			configs, apiError := api.GetConfigs(cmd, localConfig.APIHost.Value, localConfig.Token.Value, project)
			if !apiError.IsNil() {
				utils.Err(apiError.Unwrap(), apiError.Message)
			}

			var configOptions []string
			for _, val := range configs {
				configOptions = append(configOptions, val.Name)
			}
			prompt := &survey.Select{
				Message: "Select a config:",
				Options: configOptions,
			}
			err := survey.AskOne(prompt, &config)
			if err != nil {
				utils.Err(err)
			}
		}

		configuration.Set(scope, map[string]string{"project": project, "config": config})
		if !silent {
			// don't fetch the LocalConfig since we don't care about env variables or cmd flags
			conf := configuration.Get(scope)
			rows := [][]string{{"token", conf.Token.Value, conf.Token.Scope}, {"project", conf.Project.Value, conf.Project.Scope}, {"config", conf.Config.Value, conf.Config.Scope}}
			utils.PrintTable([]string{"name", "value", "scope"}, rows)
		}
	},
}

func init() {
	setupCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	setupCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	setupCmd.Flags().Bool("silent", false, "don't output the response")
	rootCmd.AddCommand(setupCmd)
}
