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
	"doppler-cli/models"
	"doppler-cli/utils"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get workplace settings",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.GetBoolFlag(cmd, "json")

		localConfig := configuration.LocalConfig(cmd)
		_, info := api.GetAPIWorkplaceSettings(cmd, localConfig.Key.Value)

		printSettings(info, jsonFlag)
	},
}

var settingsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update workplace settings",
	Run: func(cmd *cobra.Command, args []string) {
		name := cmd.Flag("name").Value.String()
		email := cmd.Flag("email").Value.String()

		if name == "" && email == "" {
			dopplerErrors.CommandMissingFlag(cmd)
		}

		jsonFlag := utils.GetBoolFlag(cmd, "json")
		silent := utils.GetBoolFlag(cmd, "silent")

		settings := models.WorkplaceSettings{Name: name, BillingEmail: email}

		localConfig := configuration.LocalConfig(cmd)
		_, info := api.SetAPIWorkplaceSettings(cmd, localConfig.Key.Value, settings)

		if !silent {
			printSettings(info, jsonFlag)
		}
	},
}

func init() {
	settingsUpdateCmd.Flags().String("name", "", "set the workplace's name")
	settingsUpdateCmd.Flags().String("email", "", "set the workplace's billing email")
	settingsUpdateCmd.Flags().Bool("json", false, "output json")
	settingsUpdateCmd.Flags().Bool("silent", false, "don't output the response")
	settingsCmd.AddCommand(settingsUpdateCmd)

	settingsCmd.Flags().Bool("json", false, "output json")
	rootCmd.AddCommand(settingsCmd)
}

func printSettings(settings models.WorkplaceSettings, jsonFlag bool) {
	if jsonFlag {
		resp, err := json.Marshal(settings)
		if err != nil {
			utils.Err(err)
		}

		fmt.Println(string(resp))
		return
	}

	rows := [][]string{{settings.ID, settings.Name, settings.BillingEmail}}
	utils.PrintTable([]string{"id", "name", "billing_email"}, rows)
}
