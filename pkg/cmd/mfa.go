/*
Copyright © 2023 Doppler <support@doppler.com>

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
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var mfaCommand = &cobra.Command{
	Use:   "mfa",
	Short: "Account MFA commands",
	Args:  cobra.NoArgs,
}

var mfaRecoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "Initiate an MFA recovery",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		if !yes {
			utils.Print("MFA recovery allows you to use the Doppler CLI to generate a short-lived, one-time recovery code to log into the Doppler dashboard.")
			utils.Print("The code will be sent to your email address.")
		}

		confirmed := yes || utils.ConfirmationPrompt("Initiate MFA recovery", false)
		if !confirmed {
			return
		}

		err := http.InitiateMfaRecovery(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		utils.Print("A one-time recovery code has been sent to your email address.")
	},
}

func init() {
	mfaRecoveryCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")

	mfaCommand.AddCommand(mfaRecoveryCmd)

	rootCmd.AddCommand(mfaCommand)
}
