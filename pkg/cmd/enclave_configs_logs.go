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
	"github.com/spf13/cobra"
)

var enclaveConfigsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List config audit logs",
	Args:  cobra.NoArgs,
	Run:   configsLogs,
}

var enclaveConfigsLogsGetCmd = &cobra.Command{
	Use:   "get [log_id]",
	Short: "Get config audit log",
	Args:  cobra.MaximumNArgs(1),
	Run:   getConfigsLogs,
}

var enclaveConfigsLogsRollbackCmd = &cobra.Command{
	Use:   "rollback [log_id]",
	Short: "Rollback a config change",
	Args:  cobra.MaximumNArgs(1),
	Run:   rollbackConfigsLogs,
}

func init() {
	enclaveConfigsLogsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsLogsCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	// TODO: hide this flag until the api supports it
	// enclaveConfigsLogsCmd.Flags().IntP("number", "n", 5, "max number of logs to display")
	enclaveConfigsCmd.AddCommand(enclaveConfigsLogsCmd)

	enclaveConfigsLogsGetCmd.Flags().String("log", "", "audit log id")
	enclaveConfigsLogsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsLogsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsLogsCmd.AddCommand(enclaveConfigsLogsGetCmd)

	enclaveConfigsLogsRollbackCmd.Flags().String("log", "", "audit log id")
	enclaveConfigsLogsRollbackCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsLogsRollbackCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsLogsCmd.AddCommand(enclaveConfigsLogsRollbackCmd)
}
