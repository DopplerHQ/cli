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
	"os"

	"github.com/DopplerHQ/cli/configuration"
	"github.com/DopplerHQ/cli/utils"
	"github.com/DopplerHQ/cli/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "doppler",
	Short: "The official Doppler CLI",
	Args:  cobra.NoArgs,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	args := os.Args[1:]
	rootCmd.ParseFlags(args)

	if rootCmd.Flags().Changed("debug") {
		utils.Debug = utils.GetBoolFlag(rootCmd, "debug")
	}
	if rootCmd.Flags().Changed("json") {
		utils.JSON = utils.GetBoolFlag(rootCmd, "json")
	}
	if rootCmd.Flags().Changed("insecure") {
		utils.Insecure = utils.GetBoolFlag(rootCmd, "insecure")
	}

	if rootCmd.Flags().Changed("configuration") {
		configuration.ConfigFile = rootCmd.Flag("configuration").Value.String()
	}
	configuration.LoadConfig()

	if err := rootCmd.Execute(); err != nil {
		utils.Err(err)
	}
}

func init() {
	rootCmd.Version = version.ProgramVersion
	rootCmd.SetVersionTemplate(rootCmd.Version + "\n")

	rootCmd.PersistentFlags().String("key", "", "doppler api key")
	rootCmd.PersistentFlags().String("api-host", "https://api.doppler.com", "api host")
	rootCmd.PersistentFlags().String("deploy-host", "https://deploy.doppler.com", "deploy host")
	rootCmd.PersistentFlags().Bool("insecure", false, "support TLS connections with invalid certificate")

	rootCmd.PersistentFlags().Bool("no-read-env", false, "don't read doppler config from the environment")
	rootCmd.PersistentFlags().String("scope", ".", "the directory to scope your config to")
	rootCmd.PersistentFlags().String("configuration", configuration.ConfigFile, "config file")
	rootCmd.PersistentFlags().Bool("json", false, "output json")
	rootCmd.PersistentFlags().Bool("debug", false, "output additional information when encountering errors")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "Get the version of the Doppler CLI")
}
