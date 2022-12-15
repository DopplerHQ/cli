/*
Copyright © 2022 Doppler <support@doppler.com>

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
	"fmt"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Manage anonymous analytics",
	Args:  cobra.NoArgs,
}

var analyticsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check whether anonymous analytics are enabled",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if utils.OutputJSON {
			printer.JSON(map[string]bool{"enabled": configuration.IsAnalyticsEnabled()})
		} else {
			adjective := "enabled"
			if !configuration.IsAnalyticsEnabled() {
				adjective = "disabled"
			}
			utils.Log(fmt.Sprintf("Anonymous analytics are currently %s", adjective))
		}
	},
}

var analyticsEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable anonymous analytics",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configuration.EnableAnalytics()

		if utils.OutputJSON {
			printer.JSON(map[string]bool{"enabled": true})
		} else {
			utils.Log("Anonymous analytics have been enabled")
		}
	},
}

var analyticsDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable anonymous analytics",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		configuration.DisableAnalytics()

		if utils.OutputJSON {
			printer.JSON(map[string]bool{"enabled": false})
		} else {
			utils.Log("Anonymous analytics have been disabled")
		}
	},
}

func init() {
	analyticsCmd.AddCommand(analyticsStatusCmd)

	analyticsCmd.AddCommand(analyticsEnableCmd)

	analyticsCmd.AddCommand(analyticsDisableCmd)

	rootCmd.AddCommand(analyticsCmd)
}
