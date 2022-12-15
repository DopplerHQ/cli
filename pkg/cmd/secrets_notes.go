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
	"os"
	"strings"

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
	var note string
	if len(args) > 1 {
		note = args[1]
	} else {
		// read from stdin
		hasData, e := utils.HasDataOnStdIn()
		if e != nil {
			utils.HandleError(e)
		}

		// note must be supplied
		if !hasData {
			utils.RequireValue("note", note)
		}

		var input []string
		scanner := bufio.NewScanner(os.Stdin)
		for {
			if ok := scanner.Scan(); !ok {
				if e := scanner.Err(); e != nil {
					utils.HandleError(e, "Unable to read input from stdin")
				}

				break
			}

			s := scanner.Text()
			input = append(input, s)
		}

		note = strings.Join(input, "\n")
	}

	utils.RequireValue("secret", secret)

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
	secretsNotesSetCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	secretsNotesCmd.AddCommand(secretsNotesSetCmd)

	secretsCmd.AddCommand(secretsNotesCmd)
}
