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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var secretsNotesCmd = &cobra.Command{
	Use:   "notes",
	Short: "Manage secret notes",
	Args:  cobra.NoArgs,
}

var secretsNotesSetCmd = &cobra.Command{
	Use:   "set [secret] [note]",
	Short: "Set a note on a secret. The secret must exist. Notes can be passed via arg or via stdin.",
	Args:  cobra.RangeArgs(1, 2),
	Run:   setSecretNote,
}

func setSecretNote(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	secret := args[0]
	utils.RequireValue("secret", secret)

	var note string
	if len(args) > 1 {
		note = args[1]
	} else {
		// read from stdin
		noteString, err := utils.GetStdIn()
		if err != nil {
			utils.HandleError(err)
		}
		if noteString == nil {
			utils.RequireValue("note", note)
		}

		note = *noteString
	}

	response, httpErr := http.SetSecretNote(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, secret, note)
	if !httpErr.IsNil() {
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	if !utils.Silent {
		printer.SecretNote(response, jsonFlag)
	}
}

func init() {
	secretsNotesSetCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	secretsNotesSetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	secretsNotesSetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsNotesSetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	secretsNotesCmd.AddCommand(secretsNotesSetCmd)

	secretsCmd.AddCommand(secretsNotesCmd)
}
