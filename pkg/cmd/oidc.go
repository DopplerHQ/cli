/*
Copyright © 2024 Doppler <support@doppler.com>

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
	"path/filepath"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var oidcCmd = &cobra.Command{
	Use:   "oidc",
	Short: "OIDC commands",
	Args:  cobra.NoArgs,
}

var oidcLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to Doppler with a service account identity via an OIDC token",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		prevConfig := configuration.Get(configuration.Scope)
		identity := cmd.Flag("identity").Value.String()
		token := cmd.Flag("token").Value.String()
		verifyTLS := utils.GetBool(localConfig.VerifyTLS.Value, true)

		utils.RequireValue("identity", identity)
		utils.RequireValue("token", token)

		// Disallow overwriting a token with the same scope
		if prevConfig.Token.Value != "" {
			prevScope, err1 := filepath.Abs(prevConfig.Token.Scope)
			newScope, err2 := filepath.Abs(configuration.Scope)
			if err1 == nil && err2 == nil && prevScope == newScope {
				if cmd.Flags().Changed("scope") {
					// user specified scope flag
					utils.PrintWarning("This scope is already authorized from a previous token. Remove the existing token to authenticate via OIDC.")
				} else {
					// scope flag wasn't specified
					utils.PrintWarning("The global scope is already authorized from a previous token. Remove the existing token to authenticate via OIDC.")
				}
				utils.Print("")
				utils.Print("Exiting")
				return
			}
		}

		response, err := http.GetOIDCAuthToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), identity, token)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		token, ok := response["token"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for auth token, expected string, got %T", response["token"]))
			utils.HandleError(errors.New("Unable to parse API response"))
		}
		expiresAt, ok := response["expires_at"].(string)
		if !ok {
			utils.LogDebug(fmt.Sprintf("Unexpected type mismatch for auth token expiration, expected string, got %T", response["expires_at"]))
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
		utils.Print(fmt.Sprintf("Authenticated via OIDC, token expires at %s", expiresAt))
	},
}

var oidcTokenRevokeCmd = &cobra.Command{
	Use:   "logout",
	Short: "Revoke the current short lived service account identity access token created via OIDC",
	Args: cobra.NoArgs,
	Run:  revokeIdentityToken,
}

func revokeIdentityToken(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)
	updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")
	updateEnclaveConfig := !utils.GetBoolFlag(cmd, "no-update-config-options")
	verifyTLS := utils.GetBool(localConfig.VerifyTLS.Value, true)
	token := localConfig.Token.Value

	utils.RequireValue("token", token)
	
	if !strings.HasPrefix(token, "dp.said.") {
		utils.PrintWarning("This command can only be used to revoke short lived service account identity tokens.")
		utils.Print("")
		utils.Print("Exiting")
		return
	}

	err := http.RevokeIdentityAuthToken(localConfig.APIHost.Value, verifyTLS, token)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	} else {
		utils.Print("Short lived OIDC token has been revoked")
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
	oidcLoginCmd.Flags().String("scope", "/", "the directory to scope your token to")
	oidcLoginCmd.Flags().String("identity", "", "the service account identity ID to authenticate as")
	oidcLoginCmd.Flags().String("token", "", "the signed OIDC JWT token string to authenticate with")

	oidcCmd.AddCommand(oidcLoginCmd)

	oidcTokenRevokeCmd.Flags().String("scope", "/", "the directory to scope your token to")
	oidcTokenRevokeCmd.Flags().Bool("no-update-config", false, "do not modify the config file")
	oidcTokenRevokeCmd.Flags().Bool("no-update-config-options", false, "do not remove configured options from the config file (i.e. project and config)")

	oidcCmd.AddCommand(oidcTokenRevokeCmd)

	rootCmd.AddCommand(oidcCmd)
}
