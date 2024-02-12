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
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var enclaveConfigsTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List a config's service tokens",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs tokens")
		configsTokens(cmd, args)
	},
}

var enclaveConfigsTokensGetCmd = &cobra.Command{
	Use:   "get [slug]",
	Short: "Get a config's service token",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs tokens get")
		getConfigsTokens(cmd, args)
	},
}

var enclaveConfigsTokensCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a service token for a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs tokens create")
		createConfigsTokens(cmd, args)
	},
}

var enclaveConfigsTokensRevokeCmd = &cobra.Command{
	Use:     "revoke [slug]",
	Aliases: []string{"delete"},
	Short:   "Revoke a service token from a config",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs tokens revoke")
		revokeConfigsTokens(cmd, args)
	},
}

func init() {
	enclaveConfigsTokensCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsTokensCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsTokensCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCmd.AddCommand(enclaveConfigsTokensCmd)

	enclaveConfigsTokensGetCmd.Flags().String("slug", "", "service token slug")
	if err := enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensGetCmd)

	enclaveConfigsTokensCreateCmd.Flags().String("name", "", "service token name")
	enclaveConfigsTokensCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsTokensCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensCreateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsTokensCreateCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensCreateCmd.Flags().Bool("plain", false, "print only the token, without formatting")
	enclaveConfigsTokensCreateCmd.Flags().Bool("copy", false, "copy the token to your clipboard")
	enclaveConfigsTokensCreateCmd.Flags().String("access", "read", "the token's access. one of [\"read\", \"read/write\"]")
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensCreateCmd)

	enclaveConfigsTokensRevokeCmd.Flags().String("slug", "", "service token slug")
	if err := enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensRevokeCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensRevokeCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensRevokeCmd)
}
