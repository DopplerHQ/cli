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
	"time"

	"github.com/DopplerHQ/cli/pkg/api"
	"github.com/DopplerHQ/cli/pkg/configuration"
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
		hostname, _ := os.Hostname()

		_, response := api.GetAPIGenerateAuthCode(cmd, localConfig.APIHost.Value, hostname, utils.HostOS(), utils.HostArch())
		code := response["code"].(string)
		authURL := response["auth_url"].(string)

		if !silent {
			fmt.Println("Your auth code is", code)
		}

		utils.CopyToClipboard(code)
		if !silent {
			fmt.Println(("This has been copied to your clipboard"))
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
		for {
			_, resp := api.GetAPIAuthToken(cmd, localConfig.APIHost.Value, code)
			if resp != nil {
				response = resp
				break
			}

			time.Sleep(2 * time.Second)
		}

		if response["success"].(bool) {
			fname := response["fname"].(string)
			token := response["token"].(string)

			configuration.Set(scope, map[string]string{"token": token, "api-host": localConfig.APIHost.Value})
			if !silent {
				fmt.Println("")
				fmt.Println("Welcome, " + fname)
			}
		} else {
			message := response["message"].(string)
			if !silent {
				fmt.Println("")
				fmt.Println(message)
			}

			os.Exit(1)
		}
	},
}

var loginRevokeCmd = &cobra.Command{
	Use:   "revoke",
	Short: "Revoke an auth token",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		silent := utils.GetBoolFlag(cmd, "silent")
		updateConfig := !utils.GetBoolFlag(cmd, "no-update-config")

		token := localConfig.Token.Value
		api.RevokeAuthToken(cmd, localConfig.APIHost.Value, token)

		if !silent {
			fmt.Println("Auth token has been revoked")
		}

		if updateConfig {
			// remove key from any configs
			for scope, config := range configuration.AllConfigs() {
				if config.Token == token {
					configuration.Set(scope, map[string]string{"token": ""})
				}
			}
		}
	},
}

func init() {
	loginCmd.Flags().Bool("silent", false, "don't output any text")
	loginCmd.Flags().String("scope", "*", "the directory to scope your token to")

	loginRevokeCmd.Flags().Bool("silent", false, "don't output any text")
	loginRevokeCmd.Flags().String("scope", "*", "the directory to scope your token to")
	loginRevokeCmd.Flags().Bool("no-update-config", false, "don't remove the revoked token from any saved configs")
	loginCmd.AddCommand(loginRevokeCmd)

	rootCmd.AddCommand(loginCmd)
}
