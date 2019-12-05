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
	"github.com/DopplerHQ/cli/pkg/api"
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var environmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "List Enclave environments",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)

		project := localConfig.Project.Value
		if len(args) > 0 {
			project = args[0]
		}

		info, err := api.GetEnvironments(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, project)
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}

		utils.PrintEnvironmentsInfo(info, jsonFlag)
	},
}

var environmentsGetCmd = &cobra.Command{
	Use:   "get [environment_id]",
	Short: "Get info for an environment",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.JSON
		localConfig := configuration.LocalConfig(cmd)
		environment := args[0]

		info, err := api.GetEnvironment(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.Project.Value, environment)
		if !err.IsNil() {
			utils.Err(err.Unwrap(), err.Message)
		}

		utils.PrintEnvironmentInfo(info, jsonFlag)
	},
}

func init() {
	environmentsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	environmentsCmd.AddCommand(environmentsGetCmd)

	rootCmd.AddCommand(environmentsCmd)
}
