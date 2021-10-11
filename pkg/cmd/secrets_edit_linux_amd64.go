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
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var header = strings.TrimPrefix(`
┌───────────────────────────────────────────────────────────────────────┐
│ Saving this file and exiting will apply your changes.                 │
│ Exiting without saving will cancel the operation.                     │
│                                                                       │
│ If a secret is removed from this file, it will be removed in Doppler. │
│                                                                       │
│ Secrets in this file follow the format:                               │
│                                                                       │
│ SECRET_ONE                                                            │
│ the secret value, which can span                                      │
│ multiple lines, and is terminated by a                                │
│ blank line followed by a period.                                      │
│                                                                       │
│ .                                                                     │
│ SECRET_TWO                                                            │
│ the second secret value                                               │
│                                                                       │
│ .                                                                     │
└───────────────────────────────────────────────────────────────────────┘

`, "\n")

func editSecrets(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)
	utils.RequireValue("token", localConfig.Token.Value)

	secrets := fetchSecretsForEdit(localConfig)
	selectedSecrets := selectSecrets(secrets)
	buffer := mapToEditFormat(selectedSecrets, secrets)
	modifiedSecrets := EditInEditor(buffer)

	if buffer == modifiedSecrets {
		utils.HandleError(fmt.Errorf("File was unchanged, not writing secrets."))
	}

	secretsToSave := parseEditFormat(selectedSecrets, modifiedSecrets)

	prompt := "Secrets that will be modified:"
	for key := range secretsToSave {
		prompt = fmt.Sprintf("%s\n- %s", prompt, key)
	}
	prompt = fmt.Sprintf("%s\n\nConfirm changes:", prompt)
	if utils.ConfirmationPrompt(prompt, false) {
		saveSecrets(localConfig, secretsToSave)
		fmt.Println("Secrets have been updated.")
	}
}

func fetchSecretsForEdit(localConfig models.ScopedOptions) map[string]models.ComputedSecret {
	response, httpErr := http.GetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
	if !httpErr.IsNil() {
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	secrets, err := models.ParseSecrets(response)
	if err != nil {
		utils.HandleError(err, "Unable to parse API response")
	}

	return secrets
}

func saveSecrets(localConfig models.ScopedOptions, secrets map[string]interface{}) {
	_, httpErr := http.SetSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secrets)
	if !httpErr.IsNil() {
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}
}

func mapToEditFormat(selectedSecrets []string, secrets map[string]models.ComputedSecret) string {
	buffer := header

	for _, val := range selectedSecrets {
		buffer = fmt.Sprintf("%s%s\n%s\n\n.\n", buffer, val, secrets[val].RawValue)
	}

	return buffer
}

func parseEditFormat(selectedSecrets []string, buffer string) map[string]interface{} {
	var firstRealLine string
	scanner := bufio.NewScanner(strings.NewReader(buffer))
	for scanner.Scan() {
		text := scanner.Text()
		if strings.HasPrefix(text, "┌") || strings.HasPrefix(text, "│") || strings.HasPrefix(text, "└") || text == "" {
			continue
		}
		firstRealLine = text
		break
	}

	secretsMap := map[string]interface{}{}
	for _, secret := range selectedSecrets {
		if _, ok := secretsMap[secret]; !ok {
			// Setting an entry to nil means delete the secret. We'll do that for all selected secrets by default
			// and then apply any actual modifications on top of it.
			secretsMap[secret] = nil
		}
	}

	if len(firstRealLine) == 0 {
		return secretsMap
	}

	buffer = buffer[strings.Index(buffer, firstRealLine):]

	secrets := strings.SplitAfter(buffer, "\n\n.\n")
	for _, secret := range secrets {
		secret = strings.Trim(secret, "\n")

		secretArr := strings.SplitN(secret, "\n", 2)
		name := secretArr[0]
		if len(name) > 0 {
			value := ""
			if len(secretArr) == 2 {
				value = strings.TrimSuffix(secretArr[1], "\n\n.")
			}
			secretsMap[name] = value
		}
	}

	return secretsMap
}

func isEditableSecret(name string) bool {
	return name != "DOPPLER_CONFIG" &&
		name != "DOPPLER_ENVIRONMENT" &&
		name != "DOPPLER_PROJECT" &&
		name != "DOPPLER_ENCLAVE_PROJECT" &&
		name != "DOPPLER_ENCLAVE_ENVIRONMENT" &&
		name != "DOPPLER_ENCLAVE_CONFIG"
}

func selectSecrets(secrets map[string]models.ComputedSecret) []string {
	var options []string

	for _, val := range secrets {
		if isEditableSecret(val.Name) {
			options = append(options, val.Name)
		}
	}

	sort.Strings(options)

	return utils.MultiSelectPrompt("Select secrets:", options)
}

func EditInEditor(contents string) string {
	tmpFd, err := utils.Memfile("fileToEdit", []byte(contents))
	if err != nil {
		utils.HandleError(err, "Unable to create file descriptor")
	}

	defer unix.Close(tmpFd)

	// filepath to our newly created in-memory file descriptor
	tmpFilePath := fmt.Sprintf("/proc/self/fd/%d", tmpFd)

	env := os.Environ()
	editor := "vim"
	for _, envVar := range env {
		// key=value format
		parts := strings.SplitN(envVar, "=", 2)
		key := parts[0]
		if strings.ToUpper(key) == "EDITOR" {
			editor = parts[1]
		}
	}

	var args []string
	if strings.HasSuffix(editor, "vim") {
		args = append(args, "+set noswapfile")
	}
	args = append(args, tmpFilePath)

	cmd := exec.Command(editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		utils.HandleError(err, "Unable to launch editor")
	}

	err = cmd.Wait()
	if err != nil {
		utils.HandleError(err, "Unable to wait for editor")
	}

	content, _ := os.ReadFile(tmpFilePath)
	return string(content)
}
