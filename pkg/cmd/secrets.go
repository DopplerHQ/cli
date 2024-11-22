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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

There are several methods for setting secrets:

1) stdin (recommended)
$ echo -e 'multiline\nvalue' | doppler secrets set CERT

2) interactive stdin (recommended)
$ doppler secrets set CERT
multiline
value

.

3) one secret
$ doppler secrets set API_KEY '123'

4) multiple secrets
$ doppler secrets set API_KEY='123' DATABASE_URL='postgres:random@127.0.0.1:5432'`,
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
var validNameTransformersList = strings.Join(models.SecretsNameTransformerTypes, ", ")
var validEnvCompatNameTransformersList = strings.Join(models.SecretsEnvCompatNameTransformerTypes, ", ")
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

var validUseEnvSettings = []string{"false", "true", "override", "only"}
var validUseEnvSettingsList = strings.Join(validUseEnvSettings, ", ")
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
JSON Secret: "{\"logging\": \"info\"}"
----------------------------------

The '--use-env' flag can be used to expose environment variables to templates:
  - 'false' (default) will not expose environment variables to templates
  - 'true' will expose both environment variables and Doppler secrets to templates. If there is a collision, the Doppler secret will take precedence.
  - 'override' will expose both environment variables and Doppler secrets to templates. If there is a collision, the environment variable will take precedence.
  - 'only' will only expose environment variables to templates (and will not fetch Doppler secrets)
`,
	Args: cobra.ExactArgs(1),
	Run:  substituteSecrets,
}

func secrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	visibility := utils.GetBoolFlag(cmd, "visibility")
	valueType := utils.GetBoolFlag(cmd, "type")
	onlyNames := utils.GetBoolFlag(cmd, "only-names")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	if onlyNames {
		secretNames, err := http.GetSecretNames(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, false)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		printer.SecretsNames(secretNames, jsonFlag)
	} else {
		response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, nil, false, 0)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		secrets, parseErr := models.ParseSecrets(response)
		if parseErr != nil {
			utils.HandleError(parseErr, "Unable to parse API response")
		}

		printer.Secrets(secrets, []string{}, jsonFlag, false, raw, false, visibility, valueType)
	}
}

func getSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	plain := utils.GetBoolFlag(cmd, "plain")
	copy := utils.GetBoolFlag(cmd, "copy")
	raw := utils.GetBoolFlag(cmd, "raw")
	visibility := utils.GetBoolFlag(cmd, "visibility")
	valueType := utils.GetBoolFlag(cmd, "type")
	exitOnMissingSecret := !utils.GetBoolFlag(cmd, "no-exit-on-missing-secret")
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	var requestedSecrets []string
	if len(args) > 0 {
		requestedSecrets = args
	}
	response, err := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, requestedSecrets, false, 0)
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

	printer.Secrets(secrets, args, jsonFlag, plain, raw, copy, visibility, valueType)
}

func setSecrets(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	raw := utils.GetBoolFlag(cmd, "raw")
	canPromptUser := !utils.GetBoolFlag(cmd, "no-interactive")
	localConfig := configuration.LocalConfig(cmd)
	visibility := cmd.Flag("visibility").Value.String()
	visibilityModified := visibility != ""
	valueType := cmd.Flag("type").Value.String()
	valueTypeModified := valueType != ""

	utils.RequireValue("token", localConfig.Token.Value)

	var changeRequests []models.ChangeRequest
	changeRequests = make([]models.ChangeRequest, 0)

	var keys []string

	// if only one arg, read from stdin
	if len(args) == 1 && !strings.Contains(args[0], "=") {
		// format: 'echo "value" | doppler secrets set KEY'
		// OR
		// format: 'doppler secrets set KEY' (interactive)

		// check for existing data on stdin
		hasData, e := utils.HasDataOnStdIn()
		if e != nil {
			utils.HandleError(e)
		}
		interactiveMode := !hasData
		if interactiveMode {
			if !canPromptUser {
				utils.HandleError(errors.New("Secret value must be provided when using --no-interactive"))
			}

			utils.Print("Enter your secret value")
			utils.Print("When finished, type a newline followed by a period")
			utils.Print("Run 'doppler secrets set --help' for more information")
			utils.Print("———————————————————— START INPUT ————————————————————")
		}

		isNewline := false
		isPreviousNewline := false
		var input []string
		scanner := bufio.NewScanner(os.Stdin)
		// read input from stdin
		for {
			if ok := scanner.Scan(); !ok {
				if e := scanner.Err(); e != nil {
					utils.HandleError(e, "Unable to read input from stdin")
				}

				break
			}

			s := scanner.Text()

			if interactiveMode {
				isPreviousNewline = isNewline
				isNewline = false

				if isPreviousNewline && s == "." {
					utils.Print("————————————————————— END INPUT —————————————————————")
					break
				}

				if len(s) == 0 {
					isNewline = true
				}
			}

			input = append(input, s)
		}

		if interactiveMode {
			// remove final newline
			input = input[:len(input)-1]
		}

		key := args[0]
		value := strings.Join(input, "\n")

		keys = append(keys, key)
		changeRequest := models.ChangeRequest{
			Name:  key,
			Value: &value,
		}
		if visibilityModified {
			changeRequest.Visibility = &visibility
		}
		if valueTypeModified {
			changeRequest.ValueType = &models.SecretValueType{
				Type: valueType,
			}
		}
		changeRequests = append(changeRequests, changeRequest)
	} else if len(args) == 2 && !strings.Contains(args[0], "=") {
		// format: 'doppler secrets set KEY value'
		key := args[0]
		value := args[1]
		keys = append(keys, key)
		changeRequest := models.ChangeRequest{
			Name:  key,
			Value: &value,
		}
		if visibilityModified {
			changeRequest.Visibility = &visibility
		}
		if valueTypeModified {
			changeRequest.ValueType = &models.SecretValueType{
				Type: valueType,
			}
		}
		changeRequests = append(changeRequests, changeRequest)
	} else {
		// format: 'doppler secrets set KEY=value'
		for _, arg := range args {
			secretArr := strings.SplitN(arg, "=", 2)
			key := secretArr[0]
			keys = append(keys, key)

			changeRequest := models.ChangeRequest{
				Name: key,
			}

			if len(secretArr) < 2 {
				changeRequest.Value = nil // don't change existing value
			} else {
				changeRequest.Value = &secretArr[1]
			}
			if visibilityModified {
				changeRequest.Visibility = &visibility
			}
			if valueTypeModified {
				changeRequest.ValueType = &models.SecretValueType{
					Type: valueType,
				}
			}
			changeRequests = append(changeRequests, changeRequest)
		}
	}

	response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, nil, changeRequests)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.Secrets(response, keys, jsonFlag, false, raw, false, visibilityModified, valueTypeModified)
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
		printer.Secrets(response, []string{}, jsonFlag, false, raw, false, false, false)
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

		response, err := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets, nil)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			printer.Secrets(response, []string{}, jsonFlag, false, raw, false, false, false)
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
	dynamicSecretsTTL := utils.GetDurationFlag(cmd, "dynamic-ttl")

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

	nameTransformerString := cmd.Flag("name-transformer").Value.String()
	var nameTransformer *models.SecretsNameTransformer
	if nameTransformerString != "" {
		nameTransformer = models.SecretsNameTransformerMap[nameTransformerString]
		if nameTransformer == nil {
			utils.HandleError(fmt.Errorf("invalid name transformer. Valid transformers are %s", validNameTransformersList))
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
			fallbackPath, legacyFallbackPath = initFallbackDir(cmd, localConfig, format, nameTransformer, nil, exitOnWriteFailure)
		}
		if enableCache {
			metadataPath = controllers.MetadataFilePath(localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, nameTransformer, nil)
		}

		fallbackOpts := controllers.FallbackOptions{
			Enable:             enableFallback,
			Path:               fallbackPath,
			LegacyPath:         legacyFallbackPath,
			Readonly:           fallbackReadonly,
			Exclusive:          fallbackOnly,
			ExitOnWriteFailure: exitOnWriteFailure,
			Passphrase:         fallbackPassphrase,
		}
		secrets := controllers.FetchSecrets(localConfig, enableCache, fallbackOpts, metadataPath, nameTransformer, dynamicSecretsTTL, format, nil)

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
		_, _, body, apiError = http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, nameTransformer, "", dynamicSecretsTTL, nil)
		if !apiError.IsNil() {
			utils.HandleError(apiError.Unwrap(), apiError.Message)
		}
	}

	if !saveFile {
		utils.Print(string(body))
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

	encryptedBody, err := crypto.Encrypt(passphrase, body, "base64")
	if err != nil {
		utils.HandleError(err, "Unable to encrypt your secrets. No file has been written.")
	}

	if err := utils.WriteFile(filePath, []byte(encryptedBody), utils.RestrictedFilePerms()); err != nil {
		utils.HandleError(err, "Unable to write the secrets file")
	}

	utils.Print(fmt.Sprintf("Downloaded secrets to %s", filePath))
}

func substituteSecrets(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)

	useEnv := cmd.Flag("use-env").Value.String()
	if !slices.Contains(validUseEnvSettings, useEnv) {
		utils.HandleError(fmt.Errorf("invalid use-env option. Valid options are %s", validUseEnvSettingsList))
	}

	if useEnv != "only" {
		// No need to require a token for env-only substitution
		utils.RequireValue("token", localConfig.Token.Value)
	}

	var outputFilePath string
	var err error
	output := cmd.Flag("output").Value.String()
	if len(output) != 0 {
		outputFilePath, err = utils.GetFilePath(output)
		if err != nil {
			utils.HandleError(err, "Unable to parse output file path")
		}
	}
	secretsMap := map[string]string{}
	env := utils.ParseEnvStrings(os.Environ())

	if useEnv != "false" {
		// If use-env is not disabled entirely, include them from the beginning
		for k, v := range env {
			secretsMap[k] = v
		}
	}

	if useEnv != "only" {
		dynamicSecretsTTL := utils.GetDurationFlag(cmd, "dynamic-ttl")
		response, responseErr := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, nil, true, dynamicSecretsTTL)
		if !responseErr.IsNil() {
			utils.HandleError(responseErr.Unwrap(), responseErr.Message)
		}

		secrets, parseErr := models.ParseSecrets(response)
		if parseErr != nil {
			utils.HandleError(parseErr, "Unable to parse API response")
		}

		for _, secret := range secrets {
			if _, ok := env[secret.Name]; useEnv == "override" && ok {
				// This secret collides with an environment variable and the env var is supposed to take precedence
				continue
			}
			if secret.ComputedValue == nil {
				// By not providing a default value when ComputedValue is nil (e.g. it's a restricted secret), we default
				// to the same behavior the substituter provides if the template file contains a secret that doesn't exist.
				continue
			}
			secretsMap[secret.Name] = *secret.ComputedValue
		}
	}

	templateBody := controllers.ReadTemplateFile(args[0])
	outputString := controllers.RenderSecretsTemplate(templateBody, secretsMap)

	if outputFilePath != "" {
		err = utils.WriteFile(outputFilePath, []byte(outputString), 0600)
		if err != nil {
			utils.HandleError(err, "Unable to save rendered data to file")
		}
		utils.Print(fmt.Sprintf("Rendered data saved to %s", outputFilePath))
	} else {
		_, err = os.Stdout.WriteString(outputString)
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
	if err := secretsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.Flags().Bool("visibility", false, "include secret visibility in table output")
	secretsCmd.Flags().Bool("type", false, "include secret type in table output")
	secretsCmd.Flags().Bool("only-names", false, "only print the secret names; omit all values")

	secretsGetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := secretsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsGetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsGetCmd.Flags().Bool("plain", false, "print values without formatting")
	secretsGetCmd.Flags().Bool("copy", false, "copy the value(s) to your clipboard")
	secretsGetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsGetCmd.Flags().Bool("visibility", false, "include secret visibility in table output")
	secretsGetCmd.Flags().Bool("type", false, "include secret type in table output")
	secretsGetCmd.Flags().Bool("no-exit-on-missing-secret", false, "do not exit if unable to find a requested secret")
	secretsCmd.AddCommand(secretsGetCmd)

	secretsSetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := secretsSetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsSetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsSetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsSetCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsSetCmd.Flags().Bool("no-interactive", false, "do not allow entering secret value via interactive mode")
	secretsSetCmd.Flags().String("visibility", "", "visibility (e.g. masked, unmasked, or restricted)")
	secretsSetCmd.Flags().String("type", "", "value type (e.g. string, decimal, etc)")
	secretsCmd.AddCommand(secretsSetCmd)

	secretsUploadCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := secretsUploadCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsUploadCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsUploadCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsUploadCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsCmd.AddCommand(secretsUploadCmd)

	secretsDeleteCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := secretsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsDeleteCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsDeleteCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsDeleteCmd.Flags().Bool("raw", false, "print the raw secret value without processing variables")
	secretsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	secretsCmd.AddCommand(secretsDeleteCmd)

	secretsDownloadCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	if err := secretsDownloadCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsDownloadCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsDownloadCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsDownloadCmd.Flags().String("format", models.JSON.String(), fmt.Sprintf("output format. one of %s", validFormatList))
	secretsDownloadCmd.Flags().String("name-transformer", "", fmt.Sprintf("output name transformer. one of %v", validNameTransformersList))
	err := secretsDownloadCmd.RegisterFlagCompletionFunc("name-transformer", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return models.SecretsNameTransformerTypes, cobra.ShellCompDirectiveDefault
	})
	if err != nil {
		utils.HandleError(err)
	}
	secretsDownloadCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the secrets file. the default passphrase is computed using your current configuration.")
	secretsDownloadCmd.Flags().Bool("no-file", false, "print the response to stdout")
	secretsDownloadCmd.Flags().Duration("dynamic-ttl", 0, "(BETA) dynamic secrets will expire after specified duration, (e.g. '3h', '15m')")
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
	if err := secretsSubstituteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsSubstituteCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	if err := secretsSubstituteCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	secretsSubstituteCmd.Flags().String("use-env", "false", fmt.Sprintf("setting for how to use environment variables passed to 'doppler secrets substitute'. One of: %s (see help ext for details)", validUseEnvSettingsList))
	secretsSubstituteCmd.Flags().String("output", "", "path to the output file. by default the rendered text will be written to stdout.")
	secretsSubstituteCmd.Flags().Duration("dynamic-ttl", 0, "(BETA) dynamic secrets will expire after specified duration, (e.g. '3h', '15m')")
	secretsCmd.AddCommand(secretsSubstituteCmd)

	rootCmd.AddCommand(secretsCmd)
}
