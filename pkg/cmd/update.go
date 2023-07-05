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
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the Doppler CLI",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		force := utils.GetBoolFlag(cmd, "force")
		available, version, err := controllers.NewVersionAvailable(models.VersionCheck{})
		if err != nil {
			utils.HandleError(err, "Unable to check for CLI updates")
		}

		if !available {
			if force {
				utils.Log("Already running the latest version but proceeding anyway due to --force flag")
			} else {
				utils.Print("You are already running the latest version")
				return
			}
		}

		controllers.InstallUpdate(version.LatestVersion)
	},
}

func init() {
	updateCmd.Flags().BoolP("force", "f", false, "install the latest CLI regardless of whether there's an update available")
	rootCmd.AddCommand(updateCmd)
}
