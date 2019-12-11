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
	"fmt"
	"os"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "doppler",
	Short: "The official Doppler CLI",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		loadFlags(cmd)
		configuration.LoadConfig()

		if utils.Debug {
			printer.ScopedConfigSource(configuration.LocalConfig(cmd), "DEBUG: Active configuration", false, true)
			fmt.Println("")
		}

		// disable version checking on the "run" command
		if version.PerformVersionCheck && cmd.CalledAs() != "run" {
			silent := cmd.Flags().Changed("silent") && utils.GetBoolFlag(cmd, "silent")
			versionCheck := http.CheckCLIVersion(configuration.VersionCheck(), silent, utils.OutputJSON, utils.Debug)
			if versionCheck != (models.VersionCheck{}) {
				if version.ProgramVersion != versionCheck.LatestVersion && !silent && !utils.OutputJSON {
					fmt.Printf("Doppler CLI version %s is now available\n", versionCheck.LatestVersion)
				}

				configuration.SetVersionCheck(versionCheck)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func loadFlags(cmd *cobra.Command) {
	if cmd.Flags().Changed("debug") {
		utils.Debug = utils.GetBoolFlag(cmd, "debug")
	}
	if cmd.Flags().Changed("json") {
		utils.OutputJSON = utils.GetBoolFlag(cmd, "json")
	}
	if cmd.Flags().Changed("no-check-version") {
		version.PerformVersionCheck = !utils.GetBoolFlag(cmd, "no-check-version")
	}
	if cmd.Flags().Changed("no-timeout") {
		http.UseTimeout = !utils.GetBoolFlag(cmd, "no-timeout")
	}
	if cmd.Flags().Changed("timeout") {
		http.TimeoutDuration = utils.GetDurationFlag(cmd, "timeout")
	}
	if cmd.Flags().Changed("configuration") {
		configuration.UserConfigPath = cmd.Flag("configuration").Value.String()
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// catch any panics
	defer func() {
		if version.IsDevelopment() {
			return
		}
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "Exception: %v\n", err)
			os.Exit(1)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version.ProgramVersion
	rootCmd.SetVersionTemplate(rootCmd.Version + "\n")
	rootCmd.Flags().BoolP("version", "v", false, "Get the version of the Doppler CLI")

	rootCmd.PersistentFlags().StringP("token", "t", "", "doppler token")
	rootCmd.PersistentFlags().String("api-host", "https://api.doppler.com", "The host address for the Doppler API")
	rootCmd.PersistentFlags().String("dashboard-host", "https://doppler.com", "The host address for the Doppler Dashboard")
	rootCmd.PersistentFlags().Bool("no-check-version", !version.PerformVersionCheck, "do not check for updates to the Doppler CLI")
	rootCmd.PersistentFlags().Bool("no-verify-tls", false, "do not verify the validity of TLS certificates on HTTP requests")
	rootCmd.PersistentFlags().Bool("no-timeout", !http.UseTimeout, "do not timeout long-running requests")
	rootCmd.PersistentFlags().Duration("timeout", http.TimeoutDuration, "how long to wait for a request to complete before timing out")

	rootCmd.PersistentFlags().Bool("no-read-env", false, "do not read enclave config from the environment")
	rootCmd.PersistentFlags().String("scope", ".", "the directory to scope your config to")
	rootCmd.PersistentFlags().String("configuration", configuration.UserConfigPath, "config file")
	rootCmd.PersistentFlags().Bool("json", false, "output json")
	rootCmd.PersistentFlags().Bool("debug", false, "output additional information when encountering errors")
}
