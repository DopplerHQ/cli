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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Get workplace settings",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		info, err := http.GetWorkplaceSettings(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.Settings(info, jsonFlag)
	},
}

var settingsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update workplace settings",
	Args: func(cmd *cobra.Command, args []string) error {
		err := cobra.NoArgs(cmd, args)
		if err != nil {
			return err
		}

		// require at least one flag to be specified
		name := cmd.Flag("name").Value.String()
		email := cmd.Flag("email").Value.String()
		if name == "" && email == "" {
			return errors.New("command needs flag --name or --email")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := cmd.Flag("name").Value.String()
		email := cmd.Flag("email").Value.String()
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		settings := models.WorkplaceSettings{Name: name, BillingEmail: email}

		info, err := http.SetWorkplaceSettings(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, settings)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.Settings(info, jsonFlag)
		}
	},
}

func init() {
	settingsUpdateCmd.Flags().String("name", "", "set the workplace's name")
	settingsUpdateCmd.Flags().String("email", "", "set the workplace's billing email")
	settingsCmd.AddCommand(settingsUpdateCmd)

	rootCmd.AddCommand(settingsCmd)
}
