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
	api "doppler-cli/api"
	configuration "doppler-cli/config"
	"doppler-cli/utils"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Get workplace activity logs",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.GetBoolFlag(cmd, "json")
		localConfig := configuration.LocalConfig(cmd)

		_, activity := api.GetAPIActivityLogs(cmd, localConfig.Key.Value)

		printActivityLogs(activity, jsonFlag)
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

		_, activity := api.GetAPIActivityLog(cmd, localConfig.Key.Value, log)

		printActivityLog(activity, jsonFlag)
	},
}

func init() {
	activityGetCmd.Flags().Bool("json", false, "output json")
	activityGetCmd.Flags().String("log", "", "activity log id")
	activityCmd.AddCommand(activityGetCmd)

	activityCmd.Flags().Bool("json", false, "output json")
	rootCmd.AddCommand(activityCmd)
}

func printActivityLogs(logs []api.ActivityLog, jsonFlag bool) {
	if jsonFlag {
		resp, err := json.Marshal(logs)
		if err != nil {
			utils.Err(err)
		}

		fmt.Println(string(resp))
		return
	}

	for _, log := range logs {
		printActivityLog(log, false)
	}
}

func printActivityLog(log api.ActivityLog, jsonFlag bool) {
	if jsonFlag {
		resp, err := json.Marshal(log)
		if err != nil {
			utils.Err(err)
		}

		fmt.Println(string(resp))
		return
	}

	dateTime, err := time.Parse(time.RFC3339, log.CreatedAt)

	fmt.Println("Log " + log.ID)
	fmt.Println("User: " + log.User.Name + " <" + log.User.Email + ">")
	if err == nil {
		fmt.Println("Date: " + dateTime.In(time.Local).String())
	}
	fmt.Println("")
	fmt.Println("\t" + log.Text)
	fmt.Println("")
}
