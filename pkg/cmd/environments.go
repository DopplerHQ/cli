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
	"errors"
	"fmt"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var environmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "Manage environments",
	Args:  cobra.NoArgs,
	Run:   environments,
}

var environmentsGetCmd = &cobra.Command{
	Use:               "get [environment_id]",
	Short:             "Get info for an environment",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: configEnvironmentIDsValidArgs,
	Run:               getEnvironments,
}

var environmentsCreateCmd = &cobra.Command{
	Use:   "create [name] [slug]",
	Short: "Create an environment",
	Args:  cobra.ExactArgs(2),
	Run:   createEnvironment,
}

var environmentsDeleteCmd = &cobra.Command{
	Use:               "delete [slug]",
	Short:             "Delete an environment",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: configEnvironmentIDsValidArgs,
	Run:               deleteEnvironment,
}

var environmentsRenameCmd = &cobra.Command{
	Use:               "rename [slug]",
	Short:             "Rename an environment",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: configEnvironmentIDsValidArgs,
	Run:               renameEnvironment,
}

func environments(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)
	number := utils.GetIntFlag(cmd, "number", 16)
	page := utils.GetIntFlag(cmd, "page", 16)

	utils.RequireValue("token", localConfig.Token.Value)

	info, err := http.GetEnvironments(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, page, number)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.EnvironmentsInfo(info, jsonFlag)
}

func getEnvironments(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)
	environment := args[0]

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("environment", environment)

	info, err := http.GetEnvironment(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, environment)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.EnvironmentInfo(info, jsonFlag)
}

func configEnvironmentIDsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	ids, err := controllers.GetEnvironmentIDs(localConfig)
	if err.IsNil() {
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func createEnvironment(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	name := args[0]
	slug := args[1]

	info, err := http.CreateEnvironment(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, name, slug)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.EnvironmentInfo(info, jsonFlag)
	}
}

func deleteEnvironment(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	slug := args[0]

	prompt := "Delete environment"
	if slug != "" {
		prompt = fmt.Sprintf("%s %s", prompt, slug)
	}

	if yes || utils.ConfirmationPrompt(prompt, false) {
		err := http.DeleteEnvironment(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, slug)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			info, err := http.GetEnvironments(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, 1, 100)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			printer.EnvironmentsInfo(info, jsonFlag)
		}
	}
}

func renameEnvironment(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)
	newName := cmd.Flag("name").Value.String()
	newSlug := cmd.Flag("slug").Value.String()

	utils.RequireValue("token", localConfig.Token.Value)

	if newName == "" && newSlug == "" {
		utils.HandleError(errors.New("command requires --name or --slug"))
	}

	slug := args[0]

	prompt := "Rename environment"
	if slug != "" {
		prompt = fmt.Sprintf("%s %s", prompt, slug)
	}

	if !yes {
		if newSlug != "" {
			utils.PrintWarning("Modifying your environment's slug may break your current deploys. All configs within this environment will also be renamed.")
		}
		yes = utils.ConfirmationPrompt(prompt, false)
	}

	if yes {
		info, err := http.RenameEnvironment(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, slug, newName, newSlug)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.EnvironmentInfo(info, jsonFlag)
		}
	}
}

func init() {
	environmentsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	environmentsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	environmentsCmd.AddCommand(environmentsGetCmd)

	environmentsCreateCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	environmentsCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	environmentsCmd.AddCommand(environmentsCreateCmd)

	environmentsDeleteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	environmentsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	environmentsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	environmentsCmd.AddCommand(environmentsDeleteCmd)

	environmentsRenameCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	environmentsRenameCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	environmentsRenameCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	environmentsRenameCmd.Flags().String("name", "", "new name")
	environmentsRenameCmd.Flags().String("slug", "", "new slug")
	environmentsCmd.AddCommand(environmentsRenameCmd)

	environmentsCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	environmentsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	environmentsCmd.Flags().IntP("number", "n", 100, "max number of environments to display")
	environmentsCmd.Flags().Int("page", 1, "page to display")
	rootCmd.AddCommand(environmentsCmd)
}
