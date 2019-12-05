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
	"time"

	"github.com/DopplerHQ/cli/pkg/api"
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to Doppler",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		scope := cmd.Flag("scope").Value.String()
		silent := utils.GetBoolFlag(cmd, "silent")
		copyAuthCode := !utils.GetBoolFlag(cmd, "no-copy")
		hostname, _ := os.Hostname()

		response, err := api.GenerateAuthCode(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), hostname, utils.HostOS(), utils.HostArch())
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}
		code := response["code"].(string)
		authURL := response["auth_url"].(string)

		if !silent {
			fmt.Println("Your auth code is", code)
		}

		if copyAuthCode {
			utils.CopyToClipboard(code)
		}

		if !silent {
			fmt.Println("")
			fmt.Println("Complete login at", authURL)
		}

		if silent || utils.ConfirmationPrompt("Open this URL in your browser?", true) {
			open.Run(authURL)
		}

		if !silent {
			fmt.Println("Waiting...")
		}

		response = nil
		// TODO can we use our existing retry function here instead??
		for {
			resp, err := api.GetAuthToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), code)
			if !err.IsNil() {
				if err.Code == 409 {
					time.Sleep(2 * time.Second)
					continue
				}
				utils.Err(err.Unwrap(), err.Message)
			}

			response = resp
			break
		}

		if response == nil {
			utils.Err(errors.New("unable to authenticate"))
		}

		if err, ok := response["error"]; ok {
			if !silent {
				fmt.Println("")
				fmt.Println(err)
			}

			os.Exit(1)
		}

		token := response["token"].(string)
		name := response["name"].(string)
		dashboard := response["dashboard_url"].(string)
		configuration.SetFromConfig(scope, models.FileScopedOptions{Token: token, APIHost: localConfig.APIHost.Value,
			DashboardHost: dashboard, VerifyTLS: localConfig.VerifyTLS.Value})

		if !silent {
			fmt.Println("")
			fmt.Println("Welcome, " + name)
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
		silent := utils.GetBoolFlag(cmd, "silent")
		updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")

		oldToken := localConfig.Token.Value
		if oldToken == "" {
			if !silent {
				fmt.Println("You must provide an auth token")
			}
			os.Exit(1)
		}

		response, err := api.RollAuthToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), oldToken)
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}

		newToken := response["token"].(string)

		if updateConfig {
			// update token in config
			for scope, config := range configuration.AllConfigs() {
				if config.Token == oldToken {
					configuration.Set(scope, map[string]string{"token": newToken})
				}
			}
		}

		if !silent {
			fmt.Println("Auth token has been rolled")
		}
	},
}

var loginRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke your auth token",
	Long: `Revoke your auth token

Your auth token will be immediately revoked.
This is the CLI equivalent to logging out.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		silent := utils.GetBoolFlag(cmd, "silent")
		updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")

		token := localConfig.Token.Value
		if token == "" {
			if !silent {
				fmt.Println("You must provide an auth token")
			}
			os.Exit(1)
		}

		_, err := api.RevokeAuthToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), token)
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}

		if updateConfig {
			// remove key from config
			for scope, config := range configuration.AllConfigs() {
				if config.Token == token {
					configuration.Set(scope, map[string]string{"token": ""})
				}
			}
		}

		if !silent {
			fmt.Println("Auth token has been revoked")
		}
	},
}

func init() {
	loginCmd.Flags().Bool("silent", false, "don't output any text")
	loginCmd.Flags().Bool("no-copy", false, "don't copy the auth code to the clipboard")
	loginCmd.Flags().String("scope", "*", "the directory to scope your token to")

	loginRollCmd.Flags().Bool("silent", false, "don't output any text")
	loginRollCmd.Flags().String("scope", "*", "the directory to scope your token to")
	loginRollCmd.Flags().Bool("no-update-config", false, "don't update the rolled token in the config file")
	loginCmd.AddCommand(loginRollCmd)

	loginRevokeCmd.Flags().Bool("silent", false, "don't output any text")
	loginRevokeCmd.Flags().String("scope", "*", "the directory to scope your token to")
	loginRevokeCmd.Flags().Bool("no-update-config", false, "don't remove the revoked token from the config file")
	loginCmd.AddCommand(loginRevokeCmd)

	rootCmd.AddCommand(loginCmd)
}
