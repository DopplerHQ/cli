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
		prevConfig := configuration.Get(configuration.Scope)
		yes := utils.GetBoolFlag(cmd, "yes")
		overwrite := utils.GetBoolFlag(cmd, "overwrite")
		copyAuthCode := !utils.GetBoolFlag(cmd, "no-copy")
		hostname, _ := os.Hostname()

		// Disallow overwriting a token with the same scope (by default)
		if prevConfig.Token.Value != "" && !overwrite {
			prevScope, err1 := filepath.Abs(prevConfig.Token.Scope)
			newScope, err2 := filepath.Abs(configuration.Scope)
			if err1 == nil && err2 == nil && prevScope == newScope {
				if cmd.Flags().Changed("scope") {
					// user specified scope flag, show yes/no override prompt
					utils.PrintWarning("This scope is already authorized from a previous login.")
					utils.Print("")
					if !utils.ConfirmationPrompt("Overwrite existing login", false) {
						utils.Print("Exiting")
						return
					}
				} else {
					// scope flag wasn't specified, show option to scope to current directory
					utils.PrintWarning("You have already logged in.")
					utils.Print("")
					utils.Print("Would you like to scope your new login to the current directory, or overwrite the existing global login?")

					cwd := utils.Cwd()
					options := []string{fmt.Sprintf("Scope login to current directory (%s)", cwd), fmt.Sprintf("Overwrite global login (%s)", newScope)}
					if utils.SelectPrompt("Select an option:", options, options[0]) == options[0] {
						// use current directory as scope
						configuration.Scope = cwd
					}
				}
			}
		}

		response, err := http.GenerateAuthCode(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), hostname, utils.HostOS(), utils.HostArch())
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		code, ok := response["code"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for auth code, expected string, got %T", response["code"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}
		pollingCode, ok := response["polling_code"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for polling code, expected string, got %T", response["polling_code"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}
		authURL, ok := response["auth_url"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for auth url, expected string, got %T", response["auth_url"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}

		if copyAuthCode {
			if err := utils.CopyToClipboard(code); err != nil {
				utils.PrintWarning("Unable to copy to clipboard")
			}
		}

		openBrowser := yes || utils.Silent || utils.ConfirmationPrompt("Open the authorization page in your browser?", true)
		if openBrowser {
			if err := open.Run(authURL); err != nil {
				if utils.Silent {
					utils.HandleError(err, "Unable to launch a browser")
				}

				utils.Log("Unable to launch a browser")
				utils.LogDebugError(err)
			}
		}

		utils.Print(fmt.Sprintf("Complete authorization at %s", authURL))
		utils.Print(fmt.Sprintf("Your auth code is:\n%s\n", color.Green.Render(code)))
		utils.Print("Waiting...")

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

			resp, err := http.GetAuthToken(localConfig.APIHost.Value, verifyTLS, pollingCode)
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
			utils.HandleError(errors.New("unexpected API response"))
		}

		if err, ok := response["error"]; ok {
			utils.Print("")
			utils.Print(fmt.Sprint(err))

			os.Exit(1)
		}

		token, ok := response["token"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for token, expected string, got %T", response["token"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}
		name, ok := response["name"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for name, expected string, got %T", response["name"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}
		dashboard, ok := response["dashboard_url"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for dashboard url, expected string, got %T", response["dashboard_url"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}

		options := map[string]string{
			models.ConfigToken.String():         token,
			models.ConfigAPIHost.String():       localConfig.APIHost.Value,
			models.ConfigDashboardHost.String(): dashboard,
		}

		// only set verifytls if using non-default value
		if !verifyTLS {
			options[models.ConfigVerifyTLS.String()] = localConfig.VerifyTLS.Value
		}

		configuration.Set(configuration.Scope, options)

		utils.Print("")
		utils.Print(fmt.Sprintf("Welcome, %s", name))

		if prevConfig.Token.Value != "" {
			prevScope, err1 := filepath.Abs(prevConfig.Token.Scope)
			newScope, err2 := filepath.Abs(configuration.Scope)
			if err1 == nil && err2 == nil && prevScope == newScope {
				utils.LogDebug("Revoking previous token")
				// this is best effort; if it fails, keep running
				_, err := http.RevokeAuthToken(prevConfig.APIHost.Value, utils.GetBool(prevConfig.VerifyTLS.Value, verifyTLS), prevConfig.Token.Value)
				if !err.IsNil() {
					utils.LogDebug("Failed to revoke token")
					utils.LogDebugError(err.Unwrap())
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

		newToken, ok := response["token"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for token, expected string, got %T", response["token"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}

		if updateConfig {
			// update token in config
			for scope, config := range configuration.AllConfigs() {
				if config.Token == oldToken {
					configuration.Set(scope, map[string]string{models.ConfigToken.String(): newToken})
				}
			}
		}

		utils.Print("Auth token has been rolled")
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
	loginCmd.Flags().Bool("no-copy", false, "do not copy the auth code to the clipboard")
	loginCmd.Flags().String("scope", "/", "the directory to scope your token to")
	loginCmd.Flags().Bool("overwrite", false, "overwrite existing token if one exists")
	loginCmd.Flags().BoolP("yes", "y", false, "open browser without confirmation")

	loginRollCmd.Flags().String("scope", "/", "the directory to scope your token to")
	loginRollCmd.Flags().Bool("no-update-config", false, "do not update the rolled token in the config file")
	loginCmd.AddCommand(loginRollCmd)

	loginRevokeCmd.Flags().String("scope", "/", "the directory to scope your token to")
	loginRevokeCmd.Flags().Bool("no-update-config", false, "do not modify the config file")
	loginRevokeCmd.Flags().Bool("no-update-config-options", false, "do not remove configured options from the config file (i.e. project and config)")
	loginRevokeCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	// deprecated
	loginRevokeCmd.Flags().Bool("no-update-enclave-config", false, "do not remove the Enclave configuration from the config file")
	if err := loginRevokeCmd.Flags().MarkDeprecated("no-update-enclave-config", "please use --no-update-config-options instead"); err != nil {
		utils.HandleError(err)
	}
	if err := loginRevokeCmd.Flags().MarkHidden("no-update-enclave-config"); err != nil {
		utils.HandleError(err)
	}

	loginCmd.AddCommand(loginRevokeCmd)
	rootCmd.AddCommand(loginCmd)
}
