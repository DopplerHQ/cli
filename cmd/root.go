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
	dopplerErrors "doppler-cli/errors"
	"doppler-cli/utils"
	"fmt"
	"os"

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

		dopplerErrors.ApplicationMissingCommand(cmd)
	},
}

func Execute() {
	args := os.Args[1:]
	rootCmd.ParseFlags(args)
	if rootCmd.Flags().Changed("debug") {
		utils.Debug = utils.GetBoolFlag(rootCmd, "debug")
	}
	if rootCmd.Flags().Changed("json") {
		utils.JSON = utils.GetBoolFlag(rootCmd, "json")
	}

	if err := rootCmd.Execute(); err != nil {
		utils.Err(err, "")
	}
}

func init() {
	// cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("key", "", "doppler api key")
	// TODO support setting api-host and deploy-host in config
	rootCmd.PersistentFlags().String("api-host", "https://staging-api.doppler.com", "api host")
	rootCmd.PersistentFlags().String("deploy-host", "https://staging-deploy.doppler.com", "deploy host")

	rootCmd.PersistentFlags().Bool("enable-env", true, "support reading doppler config from the environment")
	rootCmd.PersistentFlags().String("scope", ".", "the directory to scope your config to")
	rootCmd.PersistentFlags().String("configuration", "$HOME/.doppler.yaml", "config file")
	rootCmd.PersistentFlags().Bool("json", false, "output json")
	rootCmd.PersistentFlags().Bool("debug", false, "output additional information when encountering errors")

	rootCmd.Flags().BoolP("version", "V", false, "")
}
