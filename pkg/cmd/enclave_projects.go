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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Enclave projects",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON

		localConfig := configuration.LocalConfig(cmd)
		info, err := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ProjectsInfo(info, jsonFlag)
	},
}

var projectsGetCmd = &cobra.Command{
	Use:   "get [project_id]",
	Short: "Get info for a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}

		info, err := http.GetProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ProjectInfo(info, jsonFlag)
	},
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		description := cmd.Flag("description").Value.String()

		name := cmd.Flag("name").Value.String()
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			utils.HandleError(errors.New("you must provide a name"))
		}

		localConfig := configuration.LocalConfig(cmd)
		info, err := http.CreateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, name, description)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.ProjectInfo(info, jsonFlag)
		}
	},
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete [project_id]",
	Short: "Delete a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}

		if yes || utils.ConfirmationPrompt("Delete project "+project, false) {
			err := http.DeleteProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			if !silent {
				info, err := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
				if !err.IsNil() {
					utils.HandleError(err.Unwrap(), err.Message)
				}

				printer.ProjectsInfo(info, jsonFlag)
			}
		}
	},
}

var projectsUpdateCmd = &cobra.Command{
	Use:   "update [project_id]",
	Short: "Update a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		localConfig := configuration.LocalConfig(cmd)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}

		name := cmd.Flag("name").Value.String()
		description := cmd.Flag("description").Value.String()

		info, err := http.UpdateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project, name, description)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.ProjectInfo(info, jsonFlag)
		}
	},
}

func init() {
	projectsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	projectsCmd.AddCommand(projectsGetCmd)

	projectsCreateCmd.Flags().Bool("silent", false, "do not output the response")
	projectsCreateCmd.Flags().String("name", "", "project name")
	projectsCreateCmd.Flags().String("description", "", "project description")
	projectsCmd.AddCommand(projectsCreateCmd)

	projectsDeleteCmd.Flags().Bool("silent", false, "do not output the response")
	projectsDeleteCmd.Flags().Bool("yes", false, "proceed without confirmation")
	projectsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	projectsCmd.AddCommand(projectsDeleteCmd)

	projectsUpdateCmd.Flags().Bool("silent", false, "do not output the response")
	projectsUpdateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	projectsUpdateCmd.Flags().String("name", "", "project name")
	projectsUpdateCmd.Flags().String("description", "", "project description")
	projectsUpdateCmd.MarkFlagRequired("name")
	projectsUpdateCmd.MarkFlagRequired("description")
	projectsCmd.AddCommand(projectsUpdateCmd)

	enclaveCmd.AddCommand(projectsCmd)
}
