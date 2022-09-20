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
	"fmt"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of the CLI",
	Long: `Log out of the CLI

Your auth token will be immediately revoked.
This is an alias of the "login revoke" command.`,
	Args: cobra.NoArgs,
	Run:  revokeToken,
}

func revokeToken(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)
	updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")
	updateEnclaveConfig := !utils.GetBoolFlag(cmd, "no-update-enclave-config") && !utils.GetBoolFlag(cmd, "no-update-config-options")
	verifyTLS := utils.GetBool(localConfig.VerifyTLS.Value, true)
	yes := utils.GetBoolFlag(cmd, "yes")
	token := localConfig.Token.Value

	utils.RequireValue("token", localConfig.Token.Value)

	if !yes && !utils.ConfirmationPrompt(fmt.Sprintf("Revoke auth token scoped to %s?", localConfig.Token.Scope), false) {
		utils.Log("Aborting")
		return
	}

	_, err := http.RevokeAuthToken(localConfig.APIHost.Value, verifyTLS, token)
	if !err.IsNil() {
		// ignore error if token was invalid
		invalidTokenError := err.Code >= 400 && err.Code < 500
		if invalidTokenError {
			utils.LogDebug("Failed to revoke token")
			utils.Print(err.Unwrap().Error())
		} else {
			utils.HandleError(err.Unwrap(), err.Message)
		}
	} else {
		utils.Print("Auth token has been revoked")
	}

	if updateConfig {
		// remove key from config
		for scope, config := range configuration.AllConfigs() {
			if config.Token == token {
				optionsToUnset := []string{models.ConfigToken.String()}

				if updateEnclaveConfig {
					if config.EnclaveProject != "" {
						optionsToUnset = append(optionsToUnset, models.ConfigEnclaveProject.String())
					}
					if config.EnclaveConfig != "" {
						optionsToUnset = append(optionsToUnset, models.ConfigEnclaveConfig.String())
					}
				}

				configuration.Unset(scope, optionsToUnset)
			}
		}
	}
}

func init() {
	logoutCmd.Flags().String("scope", "/", "the directory to scope your token to")
	logoutCmd.Flags().Bool("no-update-config", false, "do not modify the config file")
	logoutCmd.Flags().Bool("no-update-config-options", false, "do not remove configured options from the config file (i.e. project and config)")
	logoutCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	// deprecated
	logoutCmd.Flags().Bool("no-update-enclave-config", false, "do not remove the config and project from the config file")
	if err := logoutCmd.Flags().MarkDeprecated("no-update-enclave-config", "please use --no-update-config-options instead"); err != nil {
		utils.HandleError(err)
	}
	if err := logoutCmd.Flags().MarkHidden("no-update-enclave-config"); err != nil {
		utils.HandleError(err)
	}

	rootCmd.AddCommand(logoutCmd)
}
