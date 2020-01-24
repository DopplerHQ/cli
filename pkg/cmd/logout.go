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
	"os"

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
	silent := utils.GetBoolFlag(cmd, "silent")
	updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")
	verifyTLS := utils.GetBool(localConfig.VerifyTLS.Value, true)
	yes := utils.GetBoolFlag(cmd, "yes")
	token := localConfig.Token.Value

	if token == "" {
		if !silent {
			fmt.Println("You must provide an auth token")
		}
		os.Exit(1)
	}

	if !yes && !utils.ConfirmationPrompt(fmt.Sprintf("Revoke auth token scoped to %s?", localConfig.Token.Scope), false) {
		return
	}

	_, err := http.RevokeAuthToken(localConfig.APIHost.Value, verifyTLS, token)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if updateConfig {
		// remove key from config
		for scope, config := range configuration.AllConfigs() {
			if config.Token == token {
				configuration.Set(scope, map[string]string{models.ConfigToken.String(): ""})
			}
		}
	}

	if !silent {
		fmt.Println("Auth token has been revoked")
	}
}

func init() {
	logoutCmd.Flags().Bool("silent", false, "disable text output")
	logoutCmd.Flags().String("scope", "*", "the directory to scope your token to")
	logoutCmd.Flags().Bool("no-update-config", false, "do not remove the revoked token from the config file")
	logoutCmd.Flags().Bool("yes", false, "proceed without confirmation")
	rootCmd.AddCommand(logoutCmd)
}
