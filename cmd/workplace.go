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
	dopplerErrors "doppler-cli/errors"
	"doppler-cli/utils"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var workplaceCmd = &cobra.Command{
	Use:   "workplace",
	Short: "Get workplace info",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.GetBoolFlag(cmd, "json")

		localConfig := configuration.LocalConfig(cmd)
		_, info := api.GetAPIWorkplace(cmd, localConfig.Key.Value)

		printWorkplaceInfo(info, jsonFlag)
	},
}

var workplaceUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a workplace's info",
	Run: func(cmd *cobra.Command, args []string) {
		name := cmd.Flag("name").Value.String()
		email := cmd.Flag("email").Value.String()

		if name == "" && email == "" {
			dopplerErrors.CommandMissingFlag(cmd)
		}

		jsonFlag := utils.GetBoolFlag(cmd, "json")
		silent := utils.GetBoolFlag(cmd, "silent")

		values := api.WorkplaceInfo{Name: name, BillingEmail: email}

		localConfig := configuration.LocalConfig(cmd)
		_, info := api.SetAPIWorkplace(cmd, localConfig.Key.Value, values)

		if !silent {
			printWorkplaceInfo(info, jsonFlag)
		}
	},
}

func init() {
	workplaceUpdateCmd.Flags().String("name", "", "set the workplace's name")
	workplaceUpdateCmd.Flags().String("email", "", "set the workplace's billing email")
	workplaceUpdateCmd.Flags().Bool("json", false, "output json")
	workplaceUpdateCmd.Flags().Bool("silent", false, "don't output the response")
	workplaceCmd.AddCommand(workplaceUpdateCmd)

	workplaceCmd.Flags().Bool("json", false, "output json")
	rootCmd.AddCommand(workplaceCmd)
}

func printWorkplaceInfo(info api.WorkplaceInfo, jsonFlag bool) {
	if jsonFlag {
		resp, err := json.Marshal(info)
		if err != nil {
			utils.Err(err)
		}

		fmt.Println(string(resp))
		return
	}

	rows := [][]string{{info.ID, info.Name, info.BillingEmail}}
	utils.PrintTable([]string{"id", "name", "billing_email"}, rows)
}
