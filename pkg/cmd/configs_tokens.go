/*
Copyright © 2020 Doppler <support@doppler.com>

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
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var configsTokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "List a config's service tokens",
	Args:  cobra.NoArgs,
	Run:   configsTokens,
}

var configsTokensGetCmd = &cobra.Command{
	Use:               "get [slug]",
	Short:             "Get a config's service token",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configTokenSlugsValidArgs,
	Run:               getConfigsTokens,
}

var configsTokensCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a service token for a config",
	Args:  cobra.MaximumNArgs(1),
	Run:   createConfigsTokens,
}

var configsTokensRevokeCmd = &cobra.Command{
	Use:               "revoke [slug|token]",
	Aliases:           []string{"delete"},
	Short:             "Revoke a service token from a config",
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: configTokenSlugsValidArgs,
	Run:               revokeConfigsTokens,
}

func configsTokens(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	tokens, err := http.GetConfigServiceTokens(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigServiceTokensInfo(tokens, len(tokens), jsonFlag)
}

func getConfigsTokens(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

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
}

func createConfigsTokens(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	plain := utils.GetBoolFlag(cmd, "plain")
	copy := utils.GetBoolFlag(cmd, "copy")
	access := cmd.Flag("access").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	name := cmd.Flag("name").Value.String()
	if len(args) > 0 {
		name = args[0]
	}
	utils.RequireValue("name", name)

	maxAge := utils.GetDurationFlagIfChanged(cmd, "max-age", 0)
	if maxAge < 0 {
		utils.HandleError(errors.New("Max age must be positive or zero"))
	}
	expireAt := time.Time{}
	if maxAge > 0 {
		expireAt = time.Now().Add(maxAge)
	}

	configToken, err := http.CreateConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, name, expireAt, access)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.ConfigServiceToken(configToken, jsonFlag, plain, copy)
}

func revokeConfigsTokens(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	slugFlagUsed := cmd.Flag("slug").Value.String() != ""

	// users can revoke tokens via slug or via the raw token value (i.e. the secret)
	var slug string
	var token string
	if len(args) > 0 {
		if slugFlagUsed {
			utils.LogWarning("slug flag is ignored when arg is specified")
		}

		value := args[0]
		isSlug := utils.IsValidUUID(value)
		if isSlug {
			slug = value
		} else {
			token = value
		}
	} else if slugFlagUsed {
		slug = cmd.Flag("slug").Value.String()
	}

	utils.RequireValue("slug or token", fmt.Sprintf("%s%s", slug, token))

	err := http.DeleteConfigServiceToken(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, slug, token)
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
}

func configTokenSlugsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	slugs, err := controllers.GetConfigTokenSlugs(localConfig)
	if err.IsNil() {
		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	configsTokensCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsTokensCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsTokensCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsTokensCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsCmd.AddCommand(configsTokensCmd)

	configsTokensGetCmd.Flags().String("slug", "", "service token slug")
	configsTokensGetCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs)
	configsTokensGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsTokensGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsTokensGetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsTokensGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsTokensCmd.AddCommand(configsTokensGetCmd)

	configsTokensCreateCmd.Flags().String("name", "", "service token name")
	configsTokensCreateCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsTokensCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsTokensCreateCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsTokensCreateCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsTokensCreateCmd.Flags().Bool("plain", false, "print only the token, without formatting")
	configsTokensCreateCmd.Flags().Bool("copy", false, "copy the token to your clipboard")
	configsTokensCreateCmd.Flags().Duration("max-age", 0, "token will expire after specified duration, (e.g. '3h', '15m')")
	configsTokensCreateCmd.Flags().String("access", "read", "the token's access. one of [\"read\", \"read/write\"]")
	configsTokensCreateCmd.RegisterFlagCompletionFunc("access", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"read", "read/write"}, cobra.ShellCompDirectiveDefault
	})
	configsTokensCmd.AddCommand(configsTokensCreateCmd)

	configsTokensRevokeCmd.Flags().String("slug", "", "service token slug")
	configsTokensRevokeCmd.RegisterFlagCompletionFunc("slug", configTokenSlugsValidArgs)
	configsTokensRevokeCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	configsTokensRevokeCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	configsTokensRevokeCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	configsTokensRevokeCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	configsTokensCmd.AddCommand(configsTokensRevokeCmd)
}
