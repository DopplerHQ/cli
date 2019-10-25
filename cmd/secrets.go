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
	"doppler-cli/api"
	configuration "doppler-cli/config"
	dopplerErrors "doppler-cli/errors"
	"doppler-cli/models"
	"doppler-cli/utils"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

type secretsResponse struct {
	Variables map[string]interface{}
	Success   bool
}

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Fetch all Doppler secrets",
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		plain := utils.GetBoolFlag(cmd, "plain")
		raw := utils.GetBoolFlag(cmd, "raw")

		localConfig := configuration.LocalConfig(cmd)
		_, secrets := api.GetAPISecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value)

		printSecrets(secrets, []string{}, jsonFlag, plain, raw)
	},
}

var secretsGetCmd = &cobra.Command{
	Use:   "get [secrets]",
	Short: "Get the value of one or more secrets",
	Long: `Get the value of one or more secrets.

Ex: output the secrets "api_key" and "crypto_key":
doppler secrets get api_key crypto_key`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		jsonFlag := utils.JSON
		plain := utils.GetBoolFlag(cmd, "plain")
		raw := utils.GetBoolFlag(cmd, "raw")

		localConfig := configuration.LocalConfig(cmd)
		_, secrets := api.GetAPISecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value)

		printSecrets(secrets, args, jsonFlag, plain, raw)
	},
}

var secretsSetCmd = &cobra.Command{
	Use:   "set [secrets]",
	Short: "Set the value of one or more secrets",
	Long: `Set the value of one or more secrets.

Ex: set the secrets "api_key" and "crypto_key":
doppler secrets set api_key=123 crypto_key=456`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		jsonFlag := utils.JSON
		plain := utils.GetBoolFlag(cmd, "plain")
		raw := utils.GetBoolFlag(cmd, "raw")
		silent := utils.GetBoolFlag(cmd, "silent")

		secrets := make(map[string]interface{})
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

		localConfig := configuration.LocalConfig(cmd)
		_, response := api.SetAPISecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, secrets)

		if !silent {
			printSecrets(response, keys, jsonFlag, plain, raw)
		}
	},
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete [secrets]",
	Short: "Delete the value of one or more secrets",
	Long: `Delete the value of one or more secrets.

Ex: delete the secrets "api_key" and "crypto_key":
doppler secrets delete api_key crypto_key`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		jsonFlag := utils.JSON
		plain := utils.GetBoolFlag(cmd, "plain")
		raw := utils.GetBoolFlag(cmd, "raw")
		silent := utils.GetBoolFlag(cmd, "silent")
		yes := utils.GetBoolFlag(cmd, "yes")

		if yes || utils.ConfirmationPrompt("Delete secret(s)") {
			secrets := make(map[string]interface{})
			for _, arg := range args {
				secrets[arg] = nil
			}

			localConfig := configuration.LocalConfig(cmd)
			_, response := api.SetAPISecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, secrets)

			if !silent {
				printSecrets(response, []string{}, jsonFlag, plain, raw)
			}
		}
	},
}

var secretsDownloadCmd = &cobra.Command{
	Use:   "download <filepath>",
	Short: "Download a config's .env file",
	Long: `Save your config's secrets to a .env file. The default filepath is ./doppler.env

Ex: download the file to /root and name it test.env:
doppler secrets download /root/test.env`,
	Run: func(cmd *cobra.Command, args []string) {
		metadata := utils.GetBoolFlag(cmd, "metadata")

		filePath := "./doppler.env"
		if len(args) > 0 {
			filePath = utils.GetFilePath(args[0], filePath)
		}

		localConfig := configuration.LocalConfig(cmd)
		body := api.DownloadSecrets(cmd, localConfig.Key.Value, localConfig.Project.Value, localConfig.Config.Value, metadata)

		err := ioutil.WriteFile(filePath, body, 0600)
		if err != nil {
			utils.Err(err, "")
		}
	},
}

func init() {
	secretsCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	secretsCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	secretsCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")

	secretsGetCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	secretsGetCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	secretsGetCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsGetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.AddCommand(secretsGetCmd)

	secretsSetCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	secretsSetCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	secretsSetCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsSetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsSetCmd.Flags().Bool("silent", false, "don't output the response")
	secretsCmd.AddCommand(secretsSetCmd)

	secretsDeleteCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	secretsDeleteCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	secretsDeleteCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsDeleteCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsDeleteCmd.Flags().Bool("silent", false, "don't output the response")
	secretsDeleteCmd.Flags().Bool("yes", false, "proceed without confirmation")
	secretsCmd.AddCommand(secretsDeleteCmd)

	secretsDownloadCmd.Flags().String("project", "", "doppler project (e.g. backend)")
	secretsDownloadCmd.Flags().String("config", "", "doppler config (e.g. dev)")
	secretsDownloadCmd.Flags().Bool("metadata", true, "add metadata to the downloaded file (helps cache busting)")
	secretsCmd.AddCommand(secretsDownloadCmd)

	rootCmd.AddCommand(secretsCmd)
}

func printSecrets(secrets map[string]models.ComputedSecret, secretsToPrint []string, jsonFlag bool, plain bool, raw bool) {
	if len(secretsToPrint) == 0 {
		for name := range secrets {
			secretsToPrint = append(secretsToPrint, name)
		}
		sort.Strings(secretsToPrint)
	}

	if jsonFlag {
		secretsMap := make(map[string]map[string]string)
		for _, name := range secretsToPrint {
			if secrets[name] != (models.ComputedSecret{}) {
				secretsMap[name] = make(map[string]string)
				secretsMap[name]["computed"] = secrets[name].ComputedValue
				if raw {
					secretsMap[name]["raw"] = secrets[name].RawValue
				}
			}
		}

		utils.PrintJSON(secretsMap)
		return
	}

	var matchedSecrets []models.ComputedSecret
	for _, name := range secretsToPrint {
		if secrets[name] != (models.ComputedSecret{}) {
			matchedSecrets = append(matchedSecrets, secrets[name])
		}
	}

	if plain {
		sbEmpty := true
		var sb strings.Builder
		for _, secret := range matchedSecrets {
			if sbEmpty {
				sbEmpty = false
			} else {
				sb.WriteString("\n")
			}

			if raw {
				sb.WriteString(secret.RawValue)
			} else {
				sb.WriteString(secret.ComputedValue)
			}
		}

		fmt.Println(sb.String())
		return
	}

	headers := []string{"name", "value"}
	if raw {
		headers = append(headers, "raw")
	}

	var rows [][]string
	for _, secret := range matchedSecrets {
		row := []string{secret.Name, secret.ComputedValue}
		if raw {
			row = append(row, secret.RawValue)
		}

		rows = append(rows, row)
	}

	utils.PrintTable(headers, rows)
}
