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
	"fmt"
	"os/exec"
	"regexp"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update the Doppler CLI",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if installedViaBrew() {
			brewUpdate()
		} else {
			utils.HandleError(fmt.Errorf("this command is not yet implemented for your install method"))
		}
	},
}

func installedViaBrew() bool {
	// this command returns 0 (nil) if installed, 1 (err) otherwise
	command := []string{"brew", "ls", "--versions", "doppler"}
	err := exec.Command(command[0], command[1:]...).Run() // #nosec G204
	return err == nil
}

func brewUpdate() {
	fmt.Println("Updating via Homebrew...")
	command := []string{"brew", "upgrade", "--fetch-HEAD", "doppler"}
	out, err := exec.Command(command[0], command[1:]...).CombinedOutput() // #nosec G204
	strOut := string(out)
	if utils.Debug {
		fmt.Println(strOut)
	}

	if err != nil {
		utils.HandleError(err, "Failed to update the Doppler CLI")
	}

	re := regexp.MustCompile(`Upgrading dopplerhq\/doppler\/doppler (\d+\.\d+\.\d+) -> (\d+\.\d+\.\d+)`)
	if matches := re.FindStringSubmatch(strOut); matches != nil {
		if len(matches) >= 2 {
			fmt.Printf("Doppler CLI was upgraded to v%s!\n", matches[2])
		} else {
			fmt.Println("Doppler CLI was upgraded!")
		}
		return
	}

	re = regexp.MustCompile(`Warning: dopplerhq\/doppler\/doppler \d+\.\d+\.\d+ already installed`)
	if loc := re.FindStringIndex(strOut); loc != nil {
		fmt.Println("You are already running the latest version")
	}
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
