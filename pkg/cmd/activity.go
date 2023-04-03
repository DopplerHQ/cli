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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Get workplace activity logs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)
		page := utils.GetIntFlag(cmd, "page", 16)
		number := utils.GetIntFlag(cmd, "number", 16)

		utils.RequireValue("token", localConfig.Token.Value)

		activity, err := http.GetActivityLogs(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, page, number)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ActivityLogs(activity, len(activity), jsonFlag)
	},
}

var activityGetCmd = &cobra.Command{
	Use:               "get [log_id]",
	Short:             "Get workplace activity log",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: activityLogIDsValidArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		log := cmd.Flag("log").Value.String()
		if len(args) > 0 {
			log = args[0]
		}
		utils.RequireValue("log", log)

		activity, err := http.GetActivityLog(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, log)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ActivityLog(activity, jsonFlag, false)
	},
}

func activityLogIDsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	ids, err := controllers.GetActivityLogIDs(localConfig)
	if err.IsNil() {
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	activityGetCmd.Flags().String("log", "", "activity log id")
	activityGetCmd.RegisterFlagCompletionFunc("log", activityLogIDsValidArgs)
	activityCmd.AddCommand(activityGetCmd)

	activityCmd.Flags().IntP("number", "n", 20, "max number of logs to display")
	activityCmd.Flags().Int("page", 1, "log page to display")
	rootCmd.AddCommand(activityCmd)
}
