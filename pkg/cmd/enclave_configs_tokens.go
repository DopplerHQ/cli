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
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var configsTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List a config's service tokens",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigServiceTokensInfo(tokens, len(tokens), jsonFlag)
	},
}

var configsTokensGetCmd = &cobra.Command{
	Use:   "get [slug]",
	Short: "Get a config's service token",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		slug := cmd.Flag("slug").Value.String()
		if len(args) > 0 {
			slug = args[0]
		}

		tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		for _, token := range tokens {
			if token.Slug == slug {
				printer.ConfigServiceTokenInfo(token, jsonFlag)
				return
			}
		}

		utils.HandleError(errors.New("invalid service token slug"), err.Message)
	},
}

var configsTokensCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a service token for a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		plain := utils.GetBoolFlag(cmd, "plain")
		localConfig := configuration.LocalConfig(cmd)

		name := cmd.Flag("name").Value.String()
		if len(args) > 0 {
			name = args[0]
		}

		configToken, err := http.CreateConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, name)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigServiceToken(configToken, jsonFlag, plain)
	},
}

var configsTokensDeleteCmd = &cobra.Command{
	Use:   "delete [slug]",
	Short: "Delete a service token from a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		silent := utils.GetBoolFlag(cmd, "silent")
		localConfig := configuration.LocalConfig(cmd)

		slug := cmd.Flag("slug").Value.String()
		if len(args) > 0 {
			slug = args[0]
		}

		err := http.DeleteConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, slug)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			printer.ConfigServiceTokensInfo(tokens, len(tokens), jsonFlag)
		}
	},
}

func init() {
	configsTokensCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsTokensCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsCmd.AddCommand(configsTokensCmd)

	configsTokensGetCmd.Flags().String("slug", "", "service token slug")
	configsTokensGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsTokensGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsTokensCmd.AddCommand(configsTokensGetCmd)

	configsTokensCreateCmd.Flags().String("name", "", "service token name")
	configsTokensCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsTokensCreateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsTokensCreateCmd.Flags().Bool("plain", false, "print only the token, without formatting")
	configsTokensCmd.AddCommand(configsTokensCreateCmd)

	configsTokensDeleteCmd.Flags().String("slug", "", "service token slug")
	configsTokensDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	configsTokensDeleteCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	configsTokensDeleteCmd.Flags().Bool("silent", false, "disable text output")
	configsTokensCmd.AddCommand(configsTokensDeleteCmd)

	enclaveCmd.AddCommand(configsCmd)
}
