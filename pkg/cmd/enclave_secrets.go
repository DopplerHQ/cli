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

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/spf13/cobra"
)

var enclaveSecretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "List Enclave secrets",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("secrets")
		secrets(cmd, args)
	},
}

var enclaveSecretsGetCmd = &cobra.Command{
	Use:   "get [secrets]",
	Short: "Get the value of one or more secrets",
	Long: `Get the value of one or more secrets.

Ex: output the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets get API_KEY CRYPTO_KEY`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("secrets get")
		getSecrets(cmd, args)
	},
}

var enclaveSecretsSetCmd = &cobra.Command{
	Use:   "set [secrets]",
	Short: "Set the value of one or more secrets",
	Long: `Set the value of one or more secrets.

Ex: set the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets set API_KEY=123 CRYPTO_KEY=456`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("secrets set")
		setSecrets(cmd, args)
	},
}

var enclaveSecretsDeleteCmd = &cobra.Command{
	Use:   "delete [secrets]",
	Short: "Delete the value of one or more secrets",
	Long: `Delete the value of one or more secrets.

Ex: delete the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets delete API_KEY CRYPTO_KEY`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("secrets delete")
		deleteSecrets(cmd, args)
	},
}

var enclaveSecretsDownloadCmd = &cobra.Command{
	Use:   "download <filepath>",
	Short: "Download a config's secrets for later use",
	Long:  `Download your config's secrets for later use. JSON and Env format are supported.`,
	Example: `Save your secrets to /root/ encrypted in JSON format
$ doppler enclave secrets download /root/secrets.json

Save your secrets to /root/ encrypted in Env format
$ doppler enclave secrets download --format=env /root/secrets.env

Print your secrets to stdout in env format without writing to the filesystem
$ doppler enclave secrets download --format=env --no-file`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("secrets download")
		downloadSecrets(cmd, args)
	},
}

func init() {
	enclaveSecretsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveSecretsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveSecretsCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveSecretsCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveSecretsCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	enclaveSecretsCmd.Flags().Bool("visibility", false, "include secret visibility in table output")
	enclaveSecretsCmd.Flags().Bool("only-names", false, "only print the secret names; omit all values")

	enclaveSecretsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveSecretsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveSecretsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveSecretsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveSecretsGetCmd.Flags().Bool("plain", false, "print values without formatting")
	enclaveSecretsGetCmd.Flags().Bool("copy", false, "copy the value(s) to your clipboard")
	enclaveSecretsGetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	enclaveSecretsGetCmd.Flags().Bool("visibility", false, "include secret visibility in table output")
	enclaveSecretsGetCmd.Flags().Bool("no-exit-on-missing-secret", false, "do not exit if unable to find a requested secret")
	enclaveSecretsCmd.AddCommand(enclaveSecretsGetCmd)

	enclaveSecretsSetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveSecretsSetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveSecretsSetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveSecretsSetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveSecretsSetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	enclaveSecretsSetCmd.Flags().Bool("no-interactive", false, "do not allow entering secret value via interactive mode")
	enclaveSecretsCmd.AddCommand(enclaveSecretsSetCmd)

	enclaveSecretsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveSecretsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveSecretsDeleteCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveSecretsDeleteCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveSecretsDeleteCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	enclaveSecretsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveSecretsCmd.AddCommand(enclaveSecretsDeleteCmd)

	enclaveSecretsDownloadCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveSecretsDownloadCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveSecretsDownloadCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	enclaveSecretsDownloadCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	enclaveSecretsDownloadCmd.Flags().String("format", models.JSON.String(), "output format. one of [json, env]")
	enclaveSecretsDownloadCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the secrets file. the default passphrase is computed using your current configuration.")
	enclaveSecretsDownloadCmd.Flags().Bool("no-file", false, "print the response to stdout")
	enclaveSecretsDownloadCmd.Flags().String("name-transformer", "", fmt.Sprintf("output name transformer. one of %v", validNameTransformersList))
	enclaveSecretsDownloadCmd.Flags().Duration("dynamic-ttl", 0, "(BETA) dynamic secrets will expire after specified duration, (e.g. '3h', '15m')")
	// fallback flags
	enclaveSecretsDownloadCmd.Flags().String("fallback", "", "path to the fallback file. encrypted secrets are written to this file after each successful fetch. secrets will be read from this file if subsequent connections are unsuccessful.")
	enclaveSecretsDownloadCmd.Flags().Bool("no-cache", false, "disable using the fallback file to speed up fetches. the fallback file is only used when the API indicates that it's still current.")
	enclaveSecretsDownloadCmd.Flags().Bool("no-fallback", false, "disable reading and writing the fallback file")
	enclaveSecretsDownloadCmd.Flags().String("fallback-passphrase", "", "passphrase to use for encrypting the fallback file. by default the passphrase is computed using your current configuration.")
	enclaveSecretsDownloadCmd.Flags().Bool("fallback-readonly", false, "disable modifying the fallback file. secrets can still be read from the file.")
	enclaveSecretsDownloadCmd.Flags().Bool("fallback-only", false, "read all secrets directly from the fallback file, without contacting Doppler. secrets will not be updated. (implies --fallback-readonly)")
	enclaveSecretsDownloadCmd.Flags().Bool("no-exit-on-write-failure", false, "do not exit if unable to write the fallback file")
	enclaveSecretsCmd.AddCommand(enclaveSecretsDownloadCmd)

	enclaveCmd.AddCommand(enclaveSecretsCmd)
}
