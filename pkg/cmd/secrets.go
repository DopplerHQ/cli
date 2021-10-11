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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
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
	Short: "Manage secrets",
	Args:  cobra.NoArgs,
	Run:   secrets,
}

var secretsGetCmd = &cobra.Command{
	Use:   "get [secrets]",
	Short: "Get the value of one or more secrets",
	Long: `Get the value of one or more secrets.

Ex: output the secrets "API_KEY" and "CRYPTO_KEY":
doppler secrets get API_KEY CRYPTO_KEY`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: secretNamesValidArgs,
	Run:               getSecrets,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set [secrets]",
	Short: "Set the value of one or more secrets",
	Long: `Set the value of one or more secrets.

Ex: set the secrets "API_KEY" and "CRYPTO_KEY":
doppler secrets set API_KEY=123 CRYPTO_KEY=456`,
	Args: cobra.MinimumNArgs(1),
	Run:  setSecrets,
}

var secretsUploadCmd = &cobra.Command{
	Use:   "upload <filepath>",
	Short: "Upload a secrets file",
	Long: `Upload a json or env secrets file.

Ex: upload an env file:
doppler secrets upload dev.env

Ex: upload a json file:
doppler secrets upload secrets.json`,
	Args: cobra.ExactArgs(1),
	Run:  uploadSecrets,
}

var secretsDeleteCmd = &cobra.Command{
	Use:   "delete [secrets]",
	Short: "Delete the value of one or more secrets",
	Long: `Delete the value of one or more secrets.

Ex: delete the secrets "API_KEY" and "CRYPTO_KEY":
doppler secrets delete API_KEY CRYPTO_KEY`,
	Args:              cobra.MinimumNArgs(1),
	Run:               deleteSecrets,
	ValidArgsFunction: secretNamesValidArgs,
}

var validFormatList = strings.Join(models.SecretFormats, ", ")
var secretsDownloadCmd = &cobra.Command{
	Use:   "download <filepath>",
	Short: "Download a config's secrets for later use",
	Long:  fmt.Sprintf("Download your config's secrets for later use. Supported formats are %s", validFormatList),
	Example: `Save your secrets to /root/ encrypted in JSON format
$ doppler secrets download /root/secrets.json

Save your secrets to /root/ encrypted in Env format
$ doppler secrets download --format=env /root/secrets.env

Print your secrets to stdout in env format without writing to the filesystem
$ doppler secrets download --format=env --no-file`,
	Args: cobra.MaximumNArgs(1),
	Run:  downloadSecrets,
}

var secretsSubstituteCmd = &cobra.Command{
	Use:   "substitute <filepath>",
	Short: "Substitute secrets into a template file",
	Long:  "Substitute secrets into a template file. See https://golang.org/pkg/text/template/ for full syntax",
	Example: `$ cat template.yaml
{{- /* Full comment support */ -}}
host: {{.API_HOST}}
port: {{.API_PORT}}

{{if .OPTIONAL_SECRET}}
optional: {{.OPTIONAL_SECRET}}
{{- /* Only rendered if OPTIONAL_SECRET IS PRESENT */ -}}
{{end}}

{{/* tojson and fromjson have been added to support parsing JSON and stringifying values: */ -}}
Multiline: {{tojson .MULTILINE_SECRET}}
JSON Secret: {{tojson .JSON_SECRET}}
$ doppler secrets substitute template.yaml
host: 127.0.0.1
port: 8080
Multiline: "Line one\r\nLine two"
JSON Secret: "{\"logging\": \"info\"}"`,
	Args: cobra.ExactArgs(1),
	Run:  substituteSecrets,
}

var secretsEditCmd = &cobra.Command{
	Use:     "edit",
	Short:   "Edit secrets in your editor",
	Long:    "Shows a list of current secrets in the config and allows you to edit in your $EDITOR",
	Example: `$ doppler secrets edit`,
	Args:    cobra.NoArgs,
	Run:     editSecrets,
}

func secrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	onlyNames := utils.GetBoolFlag(cmd, "only-names")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

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
}

func getSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	plain := utils.GetBoolFlag(cmd, "plain")
	copy := utils.GetBoolFlag(cmd, "copy")
	raw := utils.GetBoolFlag(cmd, "raw")
	exitOnMissingSecret := !utils.GetBoolFlag(cmd, "no-exit-on-missing-secret")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}
	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		utils.HandleError(parseErr, "Unable to parse API response")
	}

	if exitOnMissingSecret && len(args) > 0 {
		var missingSecrets []string

		for _, name := range args {
			if secrets[name] == (models.ComputedSecret{}) {
				missingSecrets = append(missingSecrets, name)
			}
		}

		if len(missingSecrets) > 0 {
			pluralized := "secrets"
			if len(missingSecrets) == 1 {
				pluralized = "secret"
			}
			utils.HandleError(fmt.Errorf("Could not find requested %s: %s", pluralized, strings.Join(missingSecrets, ", ")))
		}
	}

	printer.Secrets(secrets, args, jsonFlag, plain, raw, copy)
}

func setSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	secrets := map[string]interface{}{}
	var keys []string

	if len(args) == 2 && !strings.Contains(args[0], "=") {
		// format: 'doppler secrets set KEY value'
		key := args[0]
		value := args[1]
		keys = append(keys, key)
		secrets[key] = value
	} else {
		// format: 'doppler secrets set KEY=value'
		for _, arg := range args {
			secretArr := strings.SplitN(arg, "=", 2)
			keys = append(keys, secretArr[0])
			if len(secretArr) < 2 {
				secrets[secretArr[0]] = ""
			} else {
				secrets[secretArr[0]] = secretArr[1]
			}
		}
	}

	response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.Secrets(response, keys, jsonFlag, false, raw, false)
	}
}

func uploadSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	filePath, err := utils.GetFilePath(args[0])
	if err != nil {
		utils.HandleError(err, "Unable to parse upload file path")
	}

	if !utils.Exists(filePath) {
		utils.HandleError(errors.New("Upload file does not exist"))
	}

	var file []byte
	file, err = ioutil.ReadFile(filePath) // #nosec G304
	if err != nil {
		utils.HandleError(err, "Unable to read upload file")
	}

	response, httpErr := http.UploadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, string(file))
	if !httpErr.IsNil() {
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	if !utils.Silent {
		printer.Secrets(response, []string{}, jsonFlag, false, raw, false)
	}
}

func deleteSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	if yes || utils.ConfirmationPrompt("Delete secret(s)", false) {
		secrets := map[string]interface{}{}
		for _, arg := range args {
			secrets[arg] = nil
		}

		response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.Secrets(response, []string{}, jsonFlag, false, raw, false)
		}
	}
}

func downloadSecrets(cmd *cobra.Command, args []string) {
	saveFile := !utils.GetBoolFlag(cmd, "no-file")
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	enableFallback := !utils.GetBoolFlag(cmd, "no-fallback")
	enableCache := enableFallback && !utils.GetBoolFlag(cmd, "no-cache")
	fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
	fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
	exitOnWriteFailure := !utils.GetBoolFlag(cmd, "no-exit-on-write-failure")

	utils.RequireValue("token", localConfig.Token.Value)

	formatString := cmd.Flag("format").Value.String()
	var format models.SecretsFormat
	if jsonFlag {
		format = models.JSON
	}

	if formatString != "" {
		isValid := false

		for _, val := range models.SecretsFormatList {
			if val.String() == formatString {
				format = val
				isValid = true
				break
			}
		}

		if !isValid {
			utils.HandleError(fmt.Errorf("invalid format. Valid formats are %s", validFormatList))
		}
	}

	fallbackPassphrase := getPassphrase(cmd, "fallback-passphrase", localConfig)
	if fallbackPassphrase == "" {
		utils.HandleError(errors.New("invalid fallback file passphrase"))
	}

	var body []byte
	if format == models.JSON {
		fallbackPath := ""
		legacyFallbackPath := ""
		metadataPath := ""
		if enableFallback {
			fallbackPath, legacyFallbackPath = initFallbackDir(cmd, localConfig, exitOnWriteFailure)
		}
		if enableCache {
			metadataPath = controllers.MetadataFilePath(localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
		}
		secrets := fetchSecrets(localConfig, enableCache, enableFallback, fallbackPath, legacyFallbackPath, metadataPath, fallbackReadonly, fallbackOnly, exitOnWriteFailure, fallbackPassphrase)

		var err error
		body, err = json.Marshal(secrets)
		if err != nil {
			utils.HandleError(err, "Unable to parse JSON secrets")
		}
	} else {
		// fallback file is not supported when fetching env/yaml format
		enableFallback = false
		enableCache = false
		flags := []string{"fallback", "fallback-only", "fallback-readonly", "no-exit-on-write-failure"}
		for _, flag := range flags {
			if cmd.Flags().Changed(flag) {
				utils.LogWarning(fmt.Sprintf("--%s has no effect when format is %s", flag, format))
			}
		}

		var apiError http.Error
		_, _, body, apiError = http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, "")
		if !apiError.IsNil() {
			utils.HandleError(apiError.Unwrap(), apiError.Message)
		}
	}

	if !saveFile {
		fmt.Println(string(body))
		return
	}

	var filePath string
	if len(args) > 0 {
		var err error
		filePath, err = utils.GetFilePath(args[0])
		if err != nil {
			utils.HandleError(err, "Unable to parse download file path")
		}
	} else {
		filePath = filepath.Join(".", format.OutputFile())
	}

	utils.LogDebug("Encrypting secrets")

	passphrase := getPassphrase(cmd, "passphrase", localConfig)
	if passphrase == "" {
		utils.HandleError(errors.New("invalid passphrase"))
	}

	encryptedBody, err := crypto.Encrypt(passphrase, body)
	if err != nil {
		utils.HandleError(err, "Unable to encrypt your secrets. No file has been written.")
	}

	if err := utils.WriteFile(filePath, []byte(encryptedBody), utils.RestrictedFilePerms()); err != nil {
		utils.HandleError(err, "Unable to write the secrets file")
	}

	utils.Log(fmt.Sprintf("Downloaded secrets to %s", filePath))
}

func substituteSecrets(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	filePath, err := utils.GetFilePath(args[0])
	if err != nil {
		utils.HandleError(err, "Unable to parse template file path")
	}

	var file []byte
	file, err = ioutil.ReadFile(filePath) // #nosec G304
	if err != nil {
		utils.HandleError(err, "Unable to read template file")
	}

	var outputFilePath string
	output := cmd.Flag("output").Value.String()
	if len(output) != 0 {
		outputFilePath, err = utils.GetFilePath(output)
		if err != nil {
			utils.HandleError(err, "Unable to parse output file path")
		}
	}

	funcs := map[string]interface{}{
		"tojson": func(value string) (string, error) {
			body, err := json.Marshal(value)
			if err != nil {
				return "", err
			}
			return string(body), nil
		},
		"fromjson": func(value string) (interface{}, error) {
			var result interface{}
			err = json.Unmarshal([]byte(value), &result)
			if err != nil {
				return "", err
			}
			return result, nil
		},
	}
	template, err := template.New("Secrets").Funcs(funcs).Parse(string(file))
	if err != nil {
		utils.HandleError(err, "Unable to parse template text")
	}

	response, responseErr := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
	if !responseErr.IsNil() {
		utils.HandleError(responseErr.Unwrap(), responseErr.Message)
	}

	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		utils.HandleError(parseErr, "Unable to parse API response")
	}

	secretsMap := map[string]interface{}{}
	for _, secret := range secrets {
		secretsMap[secret.Name] = secret.ComputedValue
	}

	buffer := new(strings.Builder)
	err = template.Execute(buffer, secretsMap)
	if err != nil {
		utils.HandleError(err, "Unable to render template")
	}

	if outputFilePath != "" {
		err = ioutil.WriteFile(outputFilePath, []byte(buffer.String()), 0600)
		if err != nil {
			utils.HandleError(err, "Unable to save rendered data to file")
		}
		utils.Log(fmt.Sprintf("Rendered data saved to %s", outputFilePath))
	} else {
		_, err = os.Stdout.WriteString(buffer.String())
		if err != nil {
			utils.HandleError(err, "Unable to write rendered data to stdout")
		}
	}
}

func secretNamesValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	names, err := controllers.GetSecretNames(localConfig)
	if err.IsNil() {
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	secretsCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.Flags().Bool("only-names", false, "only print the secret names; omit all values")

	secretsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsGetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsGetCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsGetCmd.Flags().Bool("copy", false, "copy the value(s) to your clipboard")
	secretsGetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsGetCmd.Flags().Bool("no-exit-on-missing-secret", false, "do not exit if unable to find a requested secret")
	secretsCmd.AddCommand(secretsGetCmd)

	secretsSetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsSetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsSetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.AddCommand(secretsSetCmd)

	secretsUploadCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsUploadCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsUploadCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.AddCommand(secretsUploadCmd)

	secretsDeleteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsDeleteCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsDeleteCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	secretsCmd.AddCommand(secretsDeleteCmd)

	secretsDownloadCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsDownloadCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsDownloadCmd.Flags().String("format", models.JSON.String(), fmt.Sprintf("output format. one of %s", validFormatList))
	secretsDownloadCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the secrets file. the default passphrase is computed using your current configuration.")
	secretsDownloadCmd.Flags().Bool("no-file", false, "print the response to stdout")
	// fallback flags
	secretsDownloadCmd.Flags().String("fallback", "", "path to the fallback file. encrypted secrets are written to this file after each successful fetch. secrets will be read from this file if subsequent connections are unsuccessful.")
	secretsDownloadCmd.Flags().Bool("no-cache", false, "disable using the fallback file to speed up fetches. the fallback file is only used when the API indicates that it's still current.")
	secretsDownloadCmd.Flags().Bool("no-fallback", false, "disable reading and writing the fallback file")
	secretsDownloadCmd.Flags().String("fallback-passphrase", "", "passphrase to use for encrypting the fallback file. by default the passphrase is computed using your current configuration.")
	secretsDownloadCmd.Flags().Bool("fallback-readonly", false, "disable modifying the fallback file. secrets can still be read from the file.")
	secretsDownloadCmd.Flags().Bool("fallback-only", false, "read all secrets directly from the fallback file, without contacting Doppler. secrets will not be updated. (implies --fallback-readonly)")
	secretsDownloadCmd.Flags().Bool("no-exit-on-write-failure", false, "do not exit if unable to write the fallback file")
	secretsCmd.AddCommand(secretsDownloadCmd)

	secretsSubstituteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsSubstituteCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsSubstituteCmd.Flags().String("output", "", "path to the output file. by default the rendered text will be written to stdout.")
	secretsCmd.AddCommand(secretsSubstituteCmd)

	secretsCmd.AddCommand(secretsEditCmd)

	rootCmd.AddCommand(secretsCmd)
}
