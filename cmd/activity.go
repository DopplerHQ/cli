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
	"github.com/DopplerHQ/cli/api"
	"github.com/DopplerHQ/cli/configuration"
	"github.com/DopplerHQ/cli/utils"
	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Get workplace activity logs",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.GetBoolFlag(cmd, "json")
		localConfig := configuration.LocalConfig(cmd)
		number := utils.GetIntFlag(cmd, "number", 16)

		_, activity := api.GetAPIActivityLogs(cmd, localConfig.APIHost.Value, localConfig.Key.Value)

		utils.PrintLogs(activity, number, jsonFlag)
	},
}

var activityGetCmd = &cobra.Command{
	Use:   "get [log_id]",
	Short: "Get workplace activity log",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.GetBoolFlag(cmd, "json")
		localConfig := configuration.LocalConfig(cmd)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}

		_, activity := api.GetAPIActivityLog(cmd, localConfig.APIHost.Value, localConfig.Key.Value, log)

		utils.PrintLog(activity, jsonFlag)
	},
}

func init() {
	activityGetCmd.Flags().String("log", "", "activity log id")
	activityCmd.AddCommand(activityGetCmd)

	activityCmd.Flags().IntP("number", "n", 5, "max number of logs to display")
	rootCmd.AddCommand(activityCmd)
}
