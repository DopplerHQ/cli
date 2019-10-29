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
	"doppler-cli/api"
	configuration "doppler-cli/config"
	"doppler-cli/utils"

	"github.com/spf13/cobra"
)

type configsResponse struct {
	Variables map[string]interface{}
	Success   bool
}

var configsCmd = &cobra.Command{
	Use:   "configs",
	Short: "List configs",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)

		_, configs := api.GetAPIConfigs(cmd, localConfig.Key.Value, localConfig.Project.Value)

		utils.PrintConfigsInfo(configs, jsonFlag)
	},
}

var configsGetCmd = &cobra.Command{
	Use:   "get [config]",
	Short: "Get info for a config",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.Config.Value
		if len(args) > 0 {
			config = args[0]
		}

		_, configInfo := api.GetAPIConfig(cmd, localConfig.Key.Value, localConfig.Project.Value, config)

		utils.PrintConfigInfo(configInfo, jsonFlag)
	},
}

var configsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a config",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		silent := utils.GetBoolFlag(cmd, "silent")
		defaults := utils.GetBoolFlag(cmd, "defaults")
		environment := cmd.Flag("environment").Value.String()

		name := cmd.Flag("name").Value.String()
		if len(args) > 0 {
			name = args[0]
		}

		localConfig := configuration.LocalConfig(cmd)
		_, info := api.CreateAPIConfig(cmd, localConfig.Key.Value, localConfig.Project.Value, name, environment, defaults)

		if !silent {
			utils.PrintConfigInfo(info, jsonFlag)
		}
	},
}

var configsDeleteCmd = &cobra.Command{
	Use:   "delete [config]",
	Short: "Delete a config",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		silent := utils.GetBoolFlag(cmd, "silent")
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.Config.Value
		if len(args) > 0 {
			config = args[0]
		}

		if yes || utils.ConfirmationPrompt("Delete config "+config) {
			api.DeleteAPIConfig(cmd, localConfig.Key.Value, localConfig.Project.Value, config)

			if !silent {
				_, configs := api.GetAPIConfigs(cmd, localConfig.Key.Value, localConfig.Project.Value)
				utils.PrintConfigsInfo(configs, jsonFlag)
			}
		}
	},
}

var configsUpdateCmd = &cobra.Command{
	Use:   "update [config]",
	Short: "Update a config",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		silent := utils.GetBoolFlag(cmd, "silent")
		name := cmd.Flag("name").Value.String()
		localConfig := configuration.LocalConfig(cmd)

		config := localConfig.Config.Value
		if len(args) > 0 {
			config = args[0]
		}

		_, info := api.UpdateAPIConfig(cmd, localConfig.Key.Value, localConfig.Project.Value, config, name)

		if !silent {
			utils.PrintConfigInfo(info, jsonFlag)
		}
	},
}

var configsLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "List config audit logs",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)
		number := utils.GetIntFlag(cmd, "number", 16)

		_, logs := api.GetAPIConfigLogs(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value)

		utils.PrintLogs(logs, number, jsonFlag)
	},
}

var configsLogsGetCmd = &cobra.Command{
	Use:   "get [log_id]",
	Short: "Get config audit log",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}

		_, configLog := api.GetAPIConfigLog(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, log)

		// TODO print diff (like node cli environments:logs:view command)
		utils.PrintLog(configLog, jsonFlag)
	},
}

var configsLogsRollbackCmd = &cobra.Command{
	Use:   "rollback [log_id]",
	Short: "Rollback a config change",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		silent := utils.GetBoolFlag(cmd, "silent")
		localConfig := configuration.LocalConfig(cmd)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}

		_, configLog := api.RollbackAPIConfigLog(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, log)

		if !silent {
			// TODO print diff (like node cli environments:logs:view command)
			utils.PrintLog(configLog, jsonFlag)
		}
	},
}

func init() {
	configsCmd.Flags().String("project", "", "doppler project (e.g. backend)")

	configsGetCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsGetCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsCmd.AddCommand(configsGetCmd)

	configsCreateCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsCreateCmd.Flags().String("name", "", "config name")
	configsCreateCmd.Flags().String("environment", "", "config environment")
	configsCreateCmd.Flags().Bool("defaults", true, "populate config with environment's default secrets")
	configsCreateCmd.Flags().Bool("silent", false, "don't output the response")
	configsCreateCmd.MarkFlagRequired("environment")
	configsCmd.AddCommand(configsCreateCmd)

	configsUpdateCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsUpdateCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsUpdateCmd.Flags().String("name", "", "config name")
	configsUpdateCmd.Flags().Bool("silent", false, "don't output the response")
	configsUpdateCmd.MarkFlagRequired("name")
	configsCmd.AddCommand(configsUpdateCmd)

	configsDeleteCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsDeleteCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsDeleteCmd.Flags().Bool("silent", false, "don't output the response")
	configsDeleteCmd.Flags().Bool("yes", false, "proceed without confirmation")
	configsCmd.AddCommand(configsDeleteCmd)

	configsLogsCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsLogsCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsLogsCmd.Flags().IntP("number", "n", 5, "max number of logs to display")
	configsCmd.AddCommand(configsLogsCmd)

	configsLogsGetCmd.Flags().String("log", "", "audit log id")
	configsLogsGetCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsLogsGetCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsLogsCmd.AddCommand(configsLogsGetCmd)

	configsLogsRollbackCmd.Flags().String("log", "", "audit log id")
	configsLogsRollbackCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	configsLogsRollbackCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	configsLogsRollbackCmd.Flags().Bool("silent", false, "don't output the response")
	configsLogsCmd.AddCommand(configsLogsRollbackCmd)

	rootCmd.AddCommand(configsCmd)
}
