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

var enclaveConfigsTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List a config's service tokens",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigServiceTokensInfo(tokens, len(tokens), jsonFlag)
	},
}

var enclaveConfigsTokensGetCmd = &cobra.Command{
	Use:   "get [slug]",
	Short: "Get a config's service token",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		slug := cmd.Flag("slug").Value.String()
		if len(args) > 0 {
			slug = args[0]
		}
		utils.RequireValue("slug", slug)

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

var enclaveConfigsTokensCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a service token for a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		plain := utils.GetBoolFlag(cmd, "plain")
		copy := utils.GetBoolFlag(cmd, "copy")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		name := cmd.Flag("name").Value.String()
		if len(args) > 0 {
			name = args[0]
		}
		utils.RequireValue("name", name)

		configToken, err := http.CreateConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, name)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.ConfigServiceToken(configToken, jsonFlag, plain, copy)
	},
}

var enclaveConfigsTokensRevokeCmd = &cobra.Command{
	Use:     "revoke [slug]",
	Aliases: []string{"delete"},
	Short:   "Revoke a service token from a config",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		slug := cmd.Flag("slug").Value.String()
		if len(args) > 0 {
			slug = args[0]
		}
		utils.RequireValue("slug", slug)

		err := http.DeleteConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, slug)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			printer.ConfigServiceTokensInfo(tokens, len(tokens), jsonFlag)
		}
	},
}

func init() {
	enclaveConfigsTokensCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsCmd.AddCommand(enclaveConfigsTokensCmd)

	enclaveConfigsTokensGetCmd.Flags().String("slug", "", "service token slug")
	enclaveConfigsTokensGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensGetCmd)

	enclaveConfigsTokensCreateCmd.Flags().String("name", "", "service token name")
	enclaveConfigsTokensCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensCreateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensCreateCmd.Flags().Bool("plain", false, "print only the token, without formatting")
	enclaveConfigsTokensCreateCmd.Flags().Bool("copy", false, "copy the token to your clipboard")
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensCreateCmd)

	enclaveConfigsTokensRevokeCmd.Flags().String("slug", "", "service token slug")
	enclaveConfigsTokensRevokeCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensRevokeCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensRevokeCmd)
}
