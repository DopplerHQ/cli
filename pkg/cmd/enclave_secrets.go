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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

type secretsResponse struct {
	Variables map[string]interface{}
	Success   bool
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "List Enclave secrets",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		raw := utils.GetBoolFlag(cmd, "raw")
		onlyNames := utils.GetBoolFlag(cmd, "only-names")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		secrets, parseErr := models.ParseSecrets(response)
		if parseErr != nil {
			utils.HandleError(parseErr, "Unable to parse API response")
		}

		if onlyNames {
			printer.SecretsNames(secrets, jsonFlag)
		} else {
			printer.Secrets(secrets, []string{}, jsonFlag, false, raw, false)
		}
	},
}

var secretsGetCmd = &cobra.Command{
	Use:   "get [secrets]",
	Short: "Get the value of one or more secrets",
	Long: `Get the value of one or more secrets.

Ex: output the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets get API_KEY CRYPTO_KEY`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		plain := utils.GetBoolFlag(cmd, "plain")
		copy := utils.GetBoolFlag(cmd, "copy")
		raw := utils.GetBoolFlag(cmd, "raw")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		secrets, parseErr := models.ParseSecrets(response)
		if parseErr != nil {
			utils.HandleError(parseErr, "Unable to parse API response")
		}

		printer.Secrets(secrets, args, jsonFlag, plain, raw, copy)
	},
}

var secretsSetCmd = &cobra.Command{
	Use:   "set [secrets]",
	Short: "Set the value of one or more secrets",
	Long: `Set the value of one or more secrets.

Ex: set the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets set API_KEY=123 CRYPTO_KEY=456`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		raw := utils.GetBoolFlag(cmd, "raw")
		silent := utils.GetBoolFlag(cmd, "silent")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		secrets := map[string]interface{}{}
		var keys []string
		for _, arg := range args {
			secretArr := strings.Split(arg, "=")
			keys = append(keys, secretArr[0])
			if len(secretArr) < 2 {
				secrets[secretArr[0]] = ""
			} else {
				secrets[secretArr[0]] = secretArr[1]
			}
		}

		response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !silent {
			printer.Secrets(response, keys, jsonFlag, false, raw, false)
		}
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete [secrets]",
	Short: "Delete the value of one or more secrets",
	Long: `Delete the value of one or more secrets.

Ex: delete the secrets "API_KEY" and "CRYPTO_KEY":
doppler enclave secrets delete API_KEY CRYPTO_KEY`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		raw := utils.GetBoolFlag(cmd, "raw")
		silent := utils.GetBoolFlag(cmd, "silent")
		yes := utils.GetBoolFlag(cmd, "yes")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		if yes || utils.ConfirmationPrompt("Delete secret(s)", false) {
			secrets := map[string]interface{}{}
			for _, arg := range args {
				secrets[arg] = nil
			}

			response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			if !silent {
				printer.Secrets(response, []string{}, jsonFlag, false, raw, false)
			}
		}
	},
}

var secretsDownloadCmd = &cobra.Command{
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
		saveFile := !utils.GetBoolFlag(cmd, "no-file")
		// don't log anything extraneous when printing secrets to stdout
		silent := !saveFile || utils.GetBoolFlag(cmd, "silent")
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)
		utils.RequireValue("project", localConfig.EnclaveProject.Value)
		utils.RequireValue("config", localConfig.EnclaveConfig.Value)

		format := cmd.Flag("format").Value.String()
		if jsonFlag {
			format = "json"
		}

		validFormats := []string{"json", "env"}
		if format != "" {
			isValid := false

			for _, val := range validFormats {
				if val == format {
					isValid = true
					break
				}
			}

			if !isValid {
				utils.HandleError(fmt.Errorf("invalid format. Valid formats are %s", strings.Join(validFormats, ", ")))
			}
		}

		body, apiError := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format == "json")
		if !apiError.IsNil() {
			utils.HandleError(apiError.Unwrap(), apiError.Message)
		}

		if !saveFile {
			fmt.Println(string(body))
			return
		}

		var filePath string
		if len(args) > 0 {
			filePath = utils.GetFilePath(args[0], "")
			if filePath == "" {
				utils.HandleError(errors.New("invalid file path"))
			}
		} else if format == "env" {
			filePath = filepath.Join(".", "doppler.env")
		} else {
			filePath = filepath.Join(".", "doppler.json")
		}

		utils.LogDebug("Encrypting Enclave secrets")
		passphrase := fmt.Sprintf("%s:%s:%s", localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		if cmd.Flags().Changed("passphrase") {
			passphrase = cmd.Flag("passphrase").Value.String()
			if passphrase == "" {
				utils.HandleError(errors.New("invalid passphrase"))
			}
		}

		encryptedBody, err := crypto.Encrypt(passphrase, body)
		if err != nil {
			utils.HandleError(err, "Unable to encrypt your secrets. No file has been written.")
		}

		err = ioutil.WriteFile(filePath, []byte(encryptedBody), 0600)
		if err != nil {
			utils.HandleError(err, "Unable to write the secrets file")
		}

		if !silent {
			fmt.Println("Downloaded secrets to " + filePath)
		}
	},
}

func init() {
	secretsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	secretsCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	secretsCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.Flags().Bool("only-names", false, "only print the secret names; omit all values")

	secretsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	secretsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	secretsGetCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsGetCmd.Flags().Bool("copy", false, "copy the value(s) to your clipboard")
	secretsGetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.AddCommand(secretsGetCmd)

	secretsSetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	secretsSetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	secretsSetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsSetCmd.Flags().Bool("silent", false, "disable text output")
	secretsCmd.AddCommand(secretsSetCmd)

	secretsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	secretsDeleteCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	secretsDeleteCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsDeleteCmd.Flags().Bool("silent", false, "disable text output")
	secretsDeleteCmd.Flags().Bool("yes", false, "proceed without confirmation")
	secretsCmd.AddCommand(secretsDeleteCmd)

	secretsDownloadCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	secretsDownloadCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	secretsDownloadCmd.Flags().String("format", "json", "output format. one of [json, env]")
	secretsDownloadCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the secrets file. the default passphrase is `$token:$project:$config`.")
	secretsDownloadCmd.Flags().Bool("no-file", false, "print the response to stdout")
	secretsDownloadCmd.Flags().Bool("silent", false, "disable text output")
	secretsCmd.AddCommand(secretsDownloadCmd)

	enclaveCmd.AddCommand(secretsCmd)
}
