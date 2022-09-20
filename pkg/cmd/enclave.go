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
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var enclaveCmd = &cobra.Command{
	Use:    "enclave",
	Short:  "Manage Enclave (deprecated)",
	Args:   cobra.NoArgs,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("")

		err := cmd.Usage()
		if err != nil {
			utils.HandleError(err, "Unable to print command usage")
		}
	},
}

func init() {
	rootCmd.AddCommand(enclaveCmd)
}
