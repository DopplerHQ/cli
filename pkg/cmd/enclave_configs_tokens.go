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
	enclaveConfigsTokensCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveConfigsTokensCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveConfigsCmd.AddCommand(enclaveConfigsTokensCmd)

	enclaveConfigsTokensGetCmd.Flags().String("slug", "", "service token slug")
	enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs)
	enclaveConfigsTokensGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveConfigsTokensGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensGetCmd)

	enclaveConfigsTokensCreateCmd.Flags().String("name", "", "service token name")
	enclaveConfigsTokensCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveConfigsTokensCreateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensCreateCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveConfigsTokensCreateCmd.Flags().Bool("plain", false, "print only the token, without formatting")
	enclaveConfigsTokensCreateCmd.Flags().Bool("copy", false, "copy the token to your clipboard")
	enclaveConfigsTokensCreateCmd.Flags().String("access", "read", "the token's access. one of [\"read\", \"read/write\"]")
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensCreateCmd)

	enclaveConfigsTokensRevokeCmd.Flags().String("slug", "", "service token slug")
	enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs)
	enclaveConfigsTokensRevokeCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveConfigsTokensRevokeCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveConfigsTokensRevokeCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveConfigsTokensCmd.AddCommand(enclaveConfigsTokensRevokeCmd)
}
