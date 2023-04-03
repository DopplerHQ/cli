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
	"github.com/spf13/cobra"
)

var enclaveEnvironmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "List Enclave environments",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("environments")
		environments(cmd, args)
	},
}

var enclaveEnvironmentsGetCmd = &cobra.Command{
	Use:   "get [environment_id]",
	Short: "Get info for an environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("environments get")
		getEnvironments(cmd, args)
	},
}

func init() {
	enclaveEnvironmentsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveEnvironmentsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveEnvironmentsCmd.AddCommand(enclaveEnvironmentsGetCmd)

	enclaveEnvironmentsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	enclaveEnvironmentsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	enclaveEnvironmentsCmd.Flags().IntP("number", "n", 100, "max number of environments to display")
	enclaveEnvironmentsCmd.Flags().Int("page", 1, "page to display")
	enclaveCmd.AddCommand(enclaveEnvironmentsCmd)
}
