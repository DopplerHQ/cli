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
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

type configsResponse struct {
	Variables map[string]interface{}
	Success   bool
}

var configsCmd = &cobra.Command{
	Use:   "configs",
	Short: "List Enclave configs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		configs, err := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigsInfo(configs, jsonFlag)
	},
}

var configsGetCmd = &cobra.Command{
	Use:   "get [config]",
	Short: "Get info for a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.EnclaveConfig.Value
		if len(args) > 0 {
			config = args[0]
		}

		configInfo, err := http.GetConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigInfo(configInfo, jsonFlag)
	},
}

var configsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		defaults := !utils.GetBoolFlag(cmd, "no-defaults")
		environment := cmd.Flag("environment").Value.String()

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

		localConfig := configuration.LocalConfig(cmd)
		info, err := http.CreateConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, name, environment, defaults)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.ConfigInfo(info, jsonFlag)
		}
	},
}

var configsDeleteCmd = &cobra.Command{
	Use:   "delete [config]",
	Short: "Delete a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.EnclaveConfig.Value
		if len(args) > 0 {
			config = args[0]
		}

		if yes || utils.ConfirmationPrompt("Delete config "+config, false) {
			err := http.DeleteConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			if !silent {
				configs, err := http.GetConfigs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value)
				if !err.IsNil() {
					utils.HandleError(err.Unwrap(), err.Message)
				}

				printer.ConfigsInfo(configs, jsonFlag)
			}
		}
	},
}

var configsUpdateCmd = &cobra.Command{
	Use:   "update [config]",
	Short: "Update a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		name := cmd.Flag("name").Value.String()
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.EnclaveConfig.Value
		if len(args) > 0 {
			config = args[0]
		}

		info, err := http.UpdateConfig(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, config, name)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.ConfigInfo(info, jsonFlag)
		}
	},
}

var configsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List config audit logs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)
		// number := utils.GetIntFlag(cmd, "number", 16)

		logs, err := http.GetConfigLogs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigLogs(logs, len(logs), jsonFlag)
	},
}

var configsLogsGetCmd = &cobra.Command{
	Use:   "get [log_id]",
	Short: "Get config audit log",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}

		configLog, err := http.GetConfigLog(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, log)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigLog(configLog, jsonFlag, true)
	},
}

var configsLogsRollbackCmd = &cobra.Command{
	Use:   "rollback [log_id]",
	Short: "Rollback a config change",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		localConfig := configuration.LocalConfig(cmd)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}

		configLog, err := http.RollbackConfigLog(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, log)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.ConfigLog(configLog, jsonFlag, true)
		}
	},
}

func init() {
	configsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")

	configsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsCmd.AddCommand(configsGetCmd)

	configsCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsCreateCmd.Flags().String("name", "", "config name")
	configsCreateCmd.Flags().StringP("environment", "e", "", "config environment")
	configsCreateCmd.Flags().Bool("no-defaults", false, "do not populate config with environment's default secrets")
	configsCreateCmd.Flags().Bool("silent", false, "do not output the response")
	configsCmd.AddCommand(configsCreateCmd)

	configsUpdateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsUpdateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsUpdateCmd.Flags().String("name", "", "config name")
	configsUpdateCmd.Flags().Bool("silent", false, "do not output the response")
	configsUpdateCmd.MarkFlagRequired("name")
	configsCmd.AddCommand(configsUpdateCmd)

	configsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsDeleteCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsDeleteCmd.Flags().Bool("silent", false, "do not output the response")
	configsDeleteCmd.Flags().Bool("yes", false, "proceed without confirmation")
	configsCmd.AddCommand(configsDeleteCmd)

	configsLogsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsLogsCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	// TODO: hide this flag until the api supports it
	// configsLogsCmd.Flags().IntP("number", "n", 5, "max number of logs to display")
	configsCmd.AddCommand(configsLogsCmd)

	configsLogsGetCmd.Flags().String("log", "", "audit log id")
	configsLogsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsLogsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsLogsCmd.AddCommand(configsLogsGetCmd)

	configsLogsRollbackCmd.Flags().String("log", "", "audit log id")
	configsLogsRollbackCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsLogsRollbackCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsLogsRollbackCmd.Flags().Bool("silent", false, "do not output the response")
	configsLogsCmd.AddCommand(configsLogsRollbackCmd)

	enclaveCmd.AddCommand(configsCmd)
}
