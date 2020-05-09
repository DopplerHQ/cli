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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to Doppler",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		scope := cmd.Flag("scope").Value.String()
		prevConfig := configuration.Get(scope)
		silent := utils.GetBoolFlag(cmd, "silent")
		copyAuthCode := !utils.GetBoolFlag(cmd, "no-copy")
		hostname, _ := os.Hostname()

		response, err := http.GenerateAuthCode(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), hostname, utils.HostOS(), utils.HostArch())
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		code := response["code"].(string)
		authURL := response["auth_url"].(string)

		utils.Log(fmt.Sprintf("Your auth code is %s", color.Green.Render(code)))

		if copyAuthCode {
			utils.CopyToClipboard(code)
		}

		utils.Log("")
		utils.Log(fmt.Sprintf("Complete login at %s", authURL))

		if silent || utils.ConfirmationPrompt("Open this URL in your browser?", true) {
			err := open.Run(authURL)
			if err != nil {
				utils.HandleError(err)
			}
		}

		utils.Log("Waiting...")

		// auth flow must complete within 5 minutes
		timeout := 5 * time.Minute
		completeBy := time.Now().Add(timeout)
		verifyTLS := utils.GetBool(localConfig.VerifyTLS.Value, true)

		response = nil
		for {
			// we do not respect --no-timeout here
			if time.Now().After(completeBy) {
				utils.HandleError(fmt.Errorf("login timed out after %d minutes", int(timeout.Minutes())))
			}

			resp, err := http.GetAuthToken(localConfig.APIHost.Value, verifyTLS, code)
			if !err.IsNil() {
				if err.Code == 409 {
					time.Sleep(2 * time.Second)
					continue
				}
				utils.HandleError(err.Unwrap(), err.Message)
			}

			response = resp
			break
		}

		if response == nil {
			utils.HandleError(errors.New("unable to authenticate"))
		}

		if err, ok := response["error"]; ok {
			utils.Log("")
			utils.Log(fmt.Sprint(err))

			os.Exit(1)
		}

		token := response["token"].(string)
		name := response["name"].(string)
		dashboard := response["dashboard_url"].(string)

		options := map[string]string{
			models.ConfigToken.String():         token,
			models.ConfigAPIHost.String():       localConfig.APIHost.Value,
			models.ConfigDashboardHost.String(): dashboard,
		}

		// only set verifytls if using non-default value
		if !verifyTLS {
			options[models.ConfigVerifyTLS.String()] = localConfig.VerifyTLS.Value
		}

		configuration.Set(scope, options)

		utils.Log("")
		utils.Log(fmt.Sprintf("Welcome, %s", name))

		if prevConfig.Token.Value != "" {
			prevScope, err1 := filepath.Abs(prevConfig.Token.Scope)
			newScope, err2 := filepath.Abs(scope)
			if err1 == nil && err2 == nil && prevScope == newScope {
				utils.LogDebug("Revoking previous token")
				_, err := http.RevokeAuthToken(prevConfig.APIHost.Value, utils.GetBool(prevConfig.VerifyTLS.Value, verifyTLS), prevConfig.Token.Value)
				if !err.IsNil() {
					utils.LogDebug("Failed to revoke token")
				} else {
					utils.LogDebug("Token successfully revoked")
				}
			}
		}
	},
}

var loginRollCmd = &cobra.Command{
	Use:   "roll",
	Short: "Roll your auth token",
	Long: `Roll your auth token

This will generate a new token and revoke the old one.
Your saved configuration will be updated.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")

		utils.RequireValue("token", localConfig.Token.Value)

		oldToken := localConfig.Token.Value

		response, err := http.RollAuthToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), oldToken)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		newToken := response["token"].(string)

		if updateConfig {
			// update token in config
			for scope, config := range configuration.AllConfigs() {
				if config.Token == oldToken {
					configuration.Set(scope, map[string]string{models.ConfigToken.String(): newToken})
				}
			}
		}

		utils.Log("Auth token has been rolled")
	},
}

var loginRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke your auth token",
	Long: `Revoke your auth token

Your auth token will be immediately revoked.
This is an alias of the "logout" command.`,
	Args: cobra.NoArgs,
	Run:  revokeToken,
}

func init() {
	loginCmd.Flags().Bool("silent", false, "disable text output")
	loginCmd.Flags().Bool("no-copy", false, "do not copy the auth code to the clipboard")
	loginCmd.Flags().String("scope", "*", "the directory to scope your token to")

	loginRollCmd.Flags().Bool("silent", false, "disable text output")
	loginRollCmd.Flags().String("scope", "*", "the directory to scope your token to")
	loginRollCmd.Flags().Bool("no-update-config", false, "do not update the rolled token in the config file")
	loginCmd.AddCommand(loginRollCmd)

	loginRevokeCmd.Flags().Bool("silent", false, "disable text output")
	loginRevokeCmd.Flags().String("scope", "*", "the directory to scope your token to")
	loginRevokeCmd.Flags().Bool("no-update-config", false, "do not remove the revoked token from the config file")
	loginRevokeCmd.Flags().Bool("yes", false, "proceed without confirmation")
	loginCmd.AddCommand(loginRevokeCmd)

	rootCmd.AddCommand(loginCmd)
}
