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
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "open a web page in your browser",
}

var openDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "open the Doppler dashboard",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		localConfig := configuration.LocalConfig(cmd)
		err := open.Run(localConfig.DashboardHost.Value)
		if err != nil {
			utils.HandleError(err)
		}
	},
}

var openStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "open the Doppler status page",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		err := open.Run("https://status.doppler.com")
		if err != nil {
			utils.HandleError(err)
		}
	},
}

var openSlackCmd = &cobra.Command{
	Use:   "slack",
	Short: "open the Doppler Slack channel",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		err := open.Run("https://doppler.com/slack")
		if err != nil {
			utils.HandleError(err)
		}
	},
}

var openGithubCmd = &cobra.Command{
	Use:   "github",
	Short: "open Doppler's GitHub to help contribute",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		err := open.Run("https://doppler.com/github")
		if err != nil {
			utils.HandleError(err)
		}
	},
}

func init() {
	openCmd.AddCommand(openDashboardCmd)
	openCmd.AddCommand(openStatusCmd)
	openCmd.AddCommand(openSlackCmd)
	openCmd.AddCommand(openGithubCmd)

	rootCmd.AddCommand(openCmd)
}
