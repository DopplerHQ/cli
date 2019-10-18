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
	"doppler-cli/errors"
	"doppler-cli/utils"
	"fmt"

	"github.com/spf13/cobra"
)

/**
TODO
	- doppler run deploy-host option not working?
	- doppler run fallback options
  - table printer and --json support
	- global --configuration option
*/

var rootCmd = &cobra.Command{
	Use:   "doppler",
	Short: "The official Doppler CLI",
	Run: func(cmd *cobra.Command, args []string) {
		version := utils.GetBoolFlag(cmd, "version")
		if version {
			fmt.Println(utils.ProgramVersion)
			return
		}

		errors.ApplicationMissingCommand(cmd)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.Err(err)
	}
}

func init() {
	// cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("key", "", "doppler api key")
	rootCmd.PersistentFlags().String("project", "", "doppler project (e.g. backend)")
	rootCmd.PersistentFlags().String("config", "", "doppler config (e.g. dev)")

	rootCmd.PersistentFlags().String("api-host", "https://api.doppler.com", "api host")
	rootCmd.PersistentFlags().String("deploy-host", "https://deploy.doppler.com", "deploy host")

	rootCmd.PersistentFlags().String("scope", ".", "the directory to scope your config to")
	rootCmd.PersistentFlags().String("configuration", "$HOME/.doppler.yaml", "config file")

	rootCmd.Flags().BoolP("version", "V", false, "")
}
