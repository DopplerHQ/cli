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
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "View the CLI's changelog",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		number := utils.GetIntFlag(cmd, "number", 16)
		jsonFlag := utils.OutputJSON

		changes, apiError := controllers.CLIChangeLog()
		if !apiError.IsNil() {
			utils.HandleError(apiError.Unwrap(), apiError.Message)
		}

		printer.ChangeLog(changes, number, jsonFlag)
	},
}

func init() {
	changelogCmd.Flags().IntP("number", "n", 3, "number of versions to show changes for")

	rootCmd.AddCommand(changelogCmd)
}
