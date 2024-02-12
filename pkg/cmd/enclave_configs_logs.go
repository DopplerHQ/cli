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

var enclaveConfigsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List config audit logs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs logs")
		configsLogs(cmd, args)
	},
}

var enclaveConfigsLogsGetCmd = &cobra.Command{
	Use:   "get [log_id]",
	Short: "Get config audit log",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs logs get")
		getConfigsLogs(cmd, args)
	},
}

var enclaveConfigsLogsRollbackCmd = &cobra.Command{
	Use:   "rollback [log_id]",
	Short: "Rollback a config change",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs logs rollback")
		rollbackConfigsLogs(cmd, args)
	},
}

func init() {
	enclaveConfigsLogsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsLogsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsLogsCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsCmd.Flags().Int("page", 1, "log page to display")
	enclaveConfigsLogsCmd.Flags().IntP("number", "n", 20, "max number of logs to display")
	enclaveConfigsCmd.AddCommand(enclaveConfigsLogsCmd)

	enclaveConfigsLogsGetCmd.Flags().String("log", "", "audit log id")
	if err := enclaveConfigsLogsGetCmd.RegisterFlagCompletionFunc("log", configLogIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsLogsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsLogsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsCmd.AddCommand(enclaveConfigsLogsGetCmd)

	enclaveConfigsLogsRollbackCmd.Flags().String("log", "", "audit log id")
	if err := enclaveConfigsLogsRollbackCmd.RegisterFlagCompletionFunc("log", configLogIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsRollbackCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsLogsRollbackCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsRollbackCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsLogsRollbackCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLogsCmd.AddCommand(enclaveConfigsLogsRollbackCmd)
}
