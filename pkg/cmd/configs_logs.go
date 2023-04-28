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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var configsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List config audit logs",
	Args:  cobra.NoArgs,
	Run:   configsLogs,
}

var configsLogsGetCmd = &cobra.Command{
	Use:               "get [log_id]",
	Short:             "Get config audit log",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configLogIDsValidArgs,
	Run:               getConfigsLogs,
}

var configsLogsRollbackCmd = &cobra.Command{
	Use:               "rollback [log_id]",
	Short:             "Rollback a config change",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configLogIDsValidArgs,
	Run:               rollbackConfigsLogs,
}

func configsLogs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)
	page := utils.GetIntFlag(cmd, "page", 16)
	number := utils.GetIntFlag(cmd, "number", 16)

	utils.RequireValue("token", localConfig.Token.Value)

	logs, err := http.GetConfigLogs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, page, number)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigLogs(logs, len(logs), jsonFlag)
}

func getConfigsLogs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	log := cmd.Flag("log").Value.String()
	if len(args) > 0 {
		log = args[0]
	}
	utils.RequireValue("log", log)

	configLog, err := http.GetConfigLog(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, log)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigLog(configLog, jsonFlag, true)
}

func rollbackConfigsLogs(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	log := cmd.Flag("log").Value.String()
	if len(args) > 0 {
		log = args[0]
	}
	utils.RequireValue("log", log)

	configLog, err := http.RollbackConfigLog(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, log)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.ConfigLog(configLog, jsonFlag, true)
	}
}

func configLogIDsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	ids, err := controllers.GetConfigLogIDs(localConfig)
	if err.IsNil() {
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	configsLogsCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsLogsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsLogsCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsLogsCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsLogsCmd.Flags().Int("page", 1, "log page to display")
	configsLogsCmd.Flags().IntP("number", "n", 20, "max number of logs to display")
	configsCmd.AddCommand(configsLogsCmd)

	configsLogsGetCmd.Flags().String("log", "", "audit log id")
	configsLogsGetCmd.RegisterFlagCompletionFunc("log", configLogIDsValidArgs)
	configsLogsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsLogsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsLogsGetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsLogsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsLogsCmd.AddCommand(configsLogsGetCmd)

	configsLogsRollbackCmd.Flags().String("log", "", "audit log id")
	configsLogsRollbackCmd.RegisterFlagCompletionFunc("log", configLogIDsValidArgs)
	configsLogsRollbackCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsLogsRollbackCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsLogsRollbackCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsLogsRollbackCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsLogsCmd.AddCommand(configsLogsRollbackCmd)
}
