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
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var enclaveProjectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Enclave projects",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("projects")
		projects(cmd, args)
	},
}

var enclaveProjectsGetCmd = &cobra.Command{
	Use:   "get [project_id]",
	Short: "Get info for a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("projects get")
		getProjects(cmd, args)
	},
}

var enclaveProjectsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("projects create")
		createProjects(cmd, args)
	},
}

var enclaveProjectsDeleteCmd = &cobra.Command{
	Use:   "delete [project_id]",
	Short: "Delete a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("projects delete")
		deleteProjects(cmd, args)
	},
}

var enclaveProjectsUpdateCmd = &cobra.Command{
	Use:   "update [project_id]",
	Short: "Update a project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("projects update")
		updateProjects(cmd, args)
	},
}

func init() {
	enclaveProjectsCmd.Flags().IntP("number", "n", 100, "max number of projects to display")
	enclaveProjectsCmd.Flags().Int("page", 1, "page to display")

	enclaveProjectsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveProjectsCmd.AddCommand(enclaveProjectsGetCmd)

	enclaveProjectsCreateCmd.Flags().String("name", "", "project name")
	enclaveProjectsCreateCmd.Flags().String("description", "", "project description")
	enclaveProjectsCmd.AddCommand(enclaveProjectsCreateCmd)

	enclaveProjectsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveProjectsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveProjectsCmd.AddCommand(enclaveProjectsDeleteCmd)

	enclaveProjectsUpdateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveProjectsUpdateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveProjectsUpdateCmd.Flags().String("name", "", "project name")
	if err := enclaveProjectsUpdateCmd.MarkFlagRequired("name"); err != nil {
		utils.HandleError(err)
	}
	enclaveProjectsUpdateCmd.Flags().String("description", "", "project description")
	enclaveProjectsUpdateCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveProjectsCmd.AddCommand(enclaveProjectsUpdateCmd)

	enclaveCmd.AddCommand(enclaveProjectsCmd)
}
