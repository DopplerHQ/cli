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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var enclaveProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Enclave projects",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		info, err := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ProjectsInfo(info, jsonFlag)
	},
}

var enclaveProjectsGetCmd = &cobra.Command{
	Use:   "get [project_id]",
	Short: "Get info for a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}
		utils.RequireValue("project", project)

		info, err := http.GetProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ProjectInfo(info, jsonFlag)
	},
}

var enclaveProjectsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		description := cmd.Flag("description").Value.String()
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("description", description)

		name := cmd.Flag("name").Value.String()
		if len(args) > 0 {
			name = args[0]
		}
		utils.RequireValue("name", name)

		info, err := http.CreateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, name, description)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.ProjectInfo(info, jsonFlag)
		}
	},
}

var enclaveProjectsDeleteCmd = &cobra.Command{
	Use:   "delete [project_id]",
	Short: "Delete a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}
		utils.RequireValue("project", project)

		if yes || utils.ConfirmationPrompt("Delete project "+project, false) {
			err := http.DeleteProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			if !utils.Silent {
				info, err := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
				if !err.IsNil() {
					utils.HandleError(err.Unwrap(), err.Message)
				}

				printer.ProjectsInfo(info, jsonFlag)
			}
		}
	},
}

var enclaveProjectsUpdateCmd = &cobra.Command{
	Use:   "update [project_id]",
	Short: "Update a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		name := cmd.Flag("name").Value.String()
		description := cmd.Flag("description").Value.String()
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("name", name)
		utils.RequireValue("description", description)

		project := localConfig.EnclaveProject.Value
		if len(args) > 0 {
			project = args[0]
		}
		utils.RequireValue("project", project)

		info, err := http.UpdateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project, name, description)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.ProjectInfo(info, jsonFlag)
		}
	},
}

func init() {
	enclaveProjectsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsCmd.AddCommand(enclaveProjectsGetCmd)

	enclaveProjectsCreateCmd.Flags().String("name", "", "project name")
	enclaveProjectsCreateCmd.Flags().String("description", "", "project description")
	enclaveProjectsCmd.AddCommand(enclaveProjectsCreateCmd)

	enclaveProjectsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveProjectsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsCmd.AddCommand(enclaveProjectsDeleteCmd)

	enclaveProjectsUpdateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsUpdateCmd.Flags().String("name", "", "project name")
	enclaveProjectsUpdateCmd.Flags().String("description", "", "project description")
	if err := enclaveProjectsUpdateCmd.MarkFlagRequired("name"); err != nil {
		utils.HandleError(err)
	}
	if err := enclaveProjectsUpdateCmd.MarkFlagRequired("description"); err != nil {
		utils.HandleError(err)
	}
	enclaveProjectsCmd.AddCommand(enclaveProjectsUpdateCmd)

	enclaveCmd.AddCommand(enclaveProjectsCmd)
}
