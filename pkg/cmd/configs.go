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
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var configsCmd = &cobra.Command{
	Use:   "configs",
	Short: "Manage configs",
	Args:  cobra.NoArgs,
	Run:   configs,
}

var configsGetCmd = &cobra.Command{
	Use:               "get [config]",
	Short:             "Get info for a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configNamesValidArgs,
	Run:               getConfigs,
}

var configsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a config",
	Args:  cobra.MaximumNArgs(1),
	Run:   createConfigs,
}

var configsDeleteCmd = &cobra.Command{
	Use:               "delete [config]",
	Short:             "Delete a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configNamesValidArgs,
	Run:               deleteConfigs,
}

var configsUpdateCmd = &cobra.Command{
	Use:               "update [config]",
	Short:             "Update a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configNamesValidArgs,
	Run:               updateConfigs,
}

var configsLockCmd = &cobra.Command{
	Use:               "lock [config]",
	Short:             "Lock a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: unlockedConfigNamesValidArgs,
	Run:               lockConfigs,
}

var configsUnlockCmd = &cobra.Command{
	Use:               "unlock [config]",
	Short:             "Unlock a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: lockedConfigNamesValidArgs,
	Run:               unlockConfigs,
}

var configsCloneCmd = &cobra.Command{
	Use:               "clone [config]",
	Short:             "Clone a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configNamesValidArgs,
	Run:               cloneConfigs,
}

func configs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	environment := cmd.Flag("environment").Value.String()
	number := utils.GetIntFlag(cmd, "number", 16)
	page := utils.GetIntFlag(cmd, "page", 16)
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	configs, err := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, environment, page, number)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigsInfo(configs, jsonFlag)
}

func getConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	configInfo, err := http.GetConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigInfo(configInfo, jsonFlag)
}

func createConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	environment := cmd.Flag("environment").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	name := cmd.Flag("name").Value.String()
	if len(args) > 0 {
		name = args[0]
	}

	if name == "" {
		utils.HandleError(errors.New("you must specify a name"))
	}

	if environment == "" && strings.Index(name, "_") != -1 {
		environment = name[0:strings.Index(name, "_")]
	}

	if environment == "" {
		utils.HandleError(errors.New("you must specify an environment"))
	}

	info, err := http.CreateConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, name, environment)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.ConfigInfo(info, jsonFlag)
	}
}

func deleteConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	prompt := "Delete config"
	if config != "" {
		prompt = fmt.Sprintf("%s %s", prompt, config)
	}

	if yes || utils.ConfirmationPrompt(prompt, false) {
		err := http.DeleteConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			configs, err := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, "", 1, 100)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			printer.ConfigsInfo(configs, jsonFlag)
		}
	}
}

func updateConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	name := cmd.Flag("name").Value.String()
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("name", name)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	if !yes {
		utils.PrintWarning("Renaming this config may break your current deploys.")
		if !utils.ConfirmationPrompt("Continue?", false) {
			utils.Log("Aborting")
			return
		}
	}

	info, err := http.UpdateConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config, name)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.ConfigInfo(info, jsonFlag)
	}
}

func lockConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	prompt := "Lock config"
	if config != "" {
		prompt = fmt.Sprintf("%s %s", prompt, config)
	}

	if yes || utils.ConfirmationPrompt(prompt, false) {
		configInfo, err := http.LockConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.ConfigInfo(configInfo, jsonFlag)
		}
	}
}

func unlockConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	prompt := "Unlock config"
	if config != "" {
		prompt = fmt.Sprintf("%s %s", prompt, config)
	}

	if yes || utils.ConfirmationPrompt(prompt, false) {
		configInfo, err := http.UnlockConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.ConfigInfo(configInfo, jsonFlag)
		}
	}
}

func cloneConfigs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)
	name := cmd.Flag("name").Value.String()

	utils.RequireValue("token", localConfig.Token.Value)

	config := localConfig.EnclaveConfig.Value
	if len(args) > 0 {
		config = args[0]
	}

	configInfo, err := http.CloneConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config, name)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.ConfigInfo(configInfo, jsonFlag)
	}
}

func configNamesValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	names, err := controllers.GetConfigNames(localConfig)
	if err.IsNil() {
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func lockedConfigNamesValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	configs, err := controllers.GetConfigs(localConfig)
	if !err.IsNil() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, config := range configs {
		if config.Locked {
			names = append(names, config.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func unlockedConfigNamesValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	configs, err := controllers.GetConfigs(localConfig)
	if !err.IsNil() {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, config := range configs {
		if !config.Locked {
			names = append(names, config.Name)
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	configsCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsCmd.Flags().StringP("environment", "e", "", "config environment")
	configsCmd.RegisterFlagCompletionFunc("environment", configEnvironmentIDsValidArgs)
	configsCmd.Flags().IntP("number", "n", 100, "max number of configs to display")
	configsCmd.Flags().Int("page", 1, "page to display")

	configsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsGetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsCmd.AddCommand(configsGetCmd)

	configsCreateCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsCreateCmd.Flags().String("name", "", "config name")
	configsCreateCmd.Flags().StringP("environment", "e", "", "config environment")
	configsCreateCmd.RegisterFlagCompletionFunc("environment", configEnvironmentIDsValidArgs)
	configsCmd.AddCommand(configsCreateCmd)

	configsUpdateCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsUpdateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsUpdateCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsUpdateCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsUpdateCmd.Flags().String("name", "", "config name")
	if err := configsUpdateCmd.MarkFlagRequired("name"); err != nil {
		utils.HandleError(err)
	}
	configsUpdateCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	configsCmd.AddCommand(configsUpdateCmd)

	configsDeleteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsDeleteCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsDeleteCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	configsCmd.AddCommand(configsDeleteCmd)

	configsLockCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsLockCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsLockCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsLockCmd.RegisterFlagCompletionFunc("config", lockedConfigNamesValidArgs)
	configsLockCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	configsCmd.AddCommand(configsLockCmd)

	configsUnlockCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsUnlockCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsUnlockCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsUnlockCmd.RegisterFlagCompletionFunc("config", unlockedConfigNamesValidArgs)
	configsUnlockCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	configsCmd.AddCommand(configsUnlockCmd)

	configsCloneCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsCloneCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsCloneCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsCloneCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsCloneCmd.Flags().String("name", "", "new config name")
	configsCmd.AddCommand(configsCloneCmd)

	rootCmd.AddCommand(configsCmd)
}
