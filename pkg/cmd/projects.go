/*
Copyright © 2020 Doppler <support@doppler.com>

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
	"fmt"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Manage projects",
	Args:  cobra.NoArgs,
	Run:   projects,
}

var projectsGetCmd = &cobra.Command{
	Use:   "get [project_id]",
	Short: "Get info for a project",
	Args:  cobra.MaximumNArgs(1),
	Run:   getProjects,
}

var projectsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a project",
	Args:  cobra.MaximumNArgs(1),
	Run:   createProjects,
}

var projectsDeleteCmd = &cobra.Command{
	Use:   "delete [project_id]",
	Short: "Delete a project",
	Args:  cobra.MaximumNArgs(1),
	Run:   deleteProjects,
}

var projectsUpdateCmd = &cobra.Command{
	Use:   "update [project_id]",
	Short: "Update a project",
	Args:  cobra.MaximumNArgs(1),
	Run:   updateProjects,
}

func projects(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	info, err := http.GetProjects(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ProjectsInfo(info, jsonFlag)
}

func getProjects(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	project := localConfig.EnclaveProject.Value
	if len(args) > 0 {
		project = args[0]
	}

	info, err := http.GetProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ProjectInfo(info, jsonFlag)
}

func createProjects(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	description := cmd.Flag("description").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

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
}

func deleteProjects(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	project := localConfig.EnclaveProject.Value
	if len(args) > 0 {
		project = args[0]
	}

	prompt := "Delete project"
	if project != "" {
		prompt = fmt.Sprintf("%s %s", prompt, project)
	}

	if yes || utils.ConfirmationPrompt(prompt, false) {
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
}

func updateProjects(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	name := cmd.Flag("name").Value.String()
	description := cmd.Flag("description").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("name", name)

	project := localConfig.EnclaveProject.Value
	if len(args) > 0 {
		project = args[0]
	}

	var info models.ProjectInfo
	var httpErr http.Error
	if cmd.Flags().Changed("description") {
		info, httpErr = http.UpdateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project, name, description)
	} else {
		info, httpErr = http.UpdateProject(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project, name)
	}
	if !httpErr.IsNil() {
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	if !utils.Silent {
		printer.ProjectInfo(info, jsonFlag)
	}
}

func init() {
	projectsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	projectsCmd.AddCommand(projectsGetCmd)

	projectsCreateCmd.Flags().String("name", "", "project name")
	projectsCreateCmd.Flags().String("description", "", "project description")
	projectsCmd.AddCommand(projectsCreateCmd)

	projectsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	projectsDeleteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	projectsCmd.AddCommand(projectsDeleteCmd)

	projectsUpdateCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	projectsUpdateCmd.Flags().String("name", "", "project name")
	projectsUpdateCmd.Flags().String("description", "", "project description")
	if err := projectsUpdateCmd.MarkFlagRequired("name"); err != nil {
		utils.HandleError(err)
	}
	projectsCmd.AddCommand(projectsUpdateCmd)

	rootCmd.AddCommand(projectsCmd)
}
