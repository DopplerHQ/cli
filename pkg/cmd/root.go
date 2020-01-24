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
	"time"

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
		configuration.Setup()
		configuration.LoadConfig()

		if utils.Debug {
			utils.LogDebug("Active configuration")
			printer.ScopedConfigSource(configuration.LocalConfig(cmd), false, true)
			fmt.Println("")
		}

		silent := utils.GetBoolFlagIfChanged(cmd, "silent", false)
		plain := utils.GetBoolFlagIfChanged(cmd, "plain", false)
		canPrintResults := utils.Debug || (!silent && !plain && !utils.OutputJSON)
		checkVersion(cmd.CalledAs(), silent, plain, canPrintResults)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

func checkVersion(command string, silent bool, plain bool, print bool) {
	// disable version checking on the "run" command
	if command == "run" {
		return
	}

	if !version.PerformVersionCheck || !print || version.IsDevelopment() {
		return
	}

	prevVersionCheck := configuration.VersionCheck()
	// don't check more often than every 24 hours
	if !time.Now().After(prevVersionCheck.CheckedAt.Add(24 * time.Hour)) {
		return
	}

	versionCheck := http.CheckCLIVersion(prevVersionCheck, silent, utils.OutputJSON, utils.Debug)
	if versionCheck == (models.VersionCheck{}) {
		return
	}

	if version.ProgramVersion != versionCheck.LatestVersion {
		fmt.Printf("Doppler CLI %s is now available\n", versionCheck.LatestVersion)
	}

	configuration.SetVersionCheck(versionCheck)
}

func loadFlags(cmd *cobra.Command) {
	configuration.UserConfigFile = utils.GetFlagIfChanged(cmd, "configuration", configuration.UserConfigFile)
	http.TimeoutDuration = utils.GetDurationFlagIfChanged(cmd, "timeout", http.TimeoutDuration)
	http.UseTimeout = !utils.GetBoolFlagIfChanged(cmd, "no-timeout", !http.UseTimeout)
	utils.Debug = utils.GetBoolFlagIfChanged(cmd, "debug", utils.Debug)
	utils.OutputJSON = utils.GetBoolFlagIfChanged(cmd, "json", utils.OutputJSON)
	version.PerformVersionCheck = !utils.GetBoolFlagIfChanged(cmd, "no-check-version", !version.PerformVersionCheck)
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
	rootCmd.PersistentFlags().Bool("no-update", !version.PerformVersionCheck, "disable checking for Doppler CLI updates")
	rootCmd.PersistentFlags().Bool("no-verify-tls", false, "do not verify the validity of TLS certificates on HTTP requests (not recommended)")
	rootCmd.PersistentFlags().Bool("no-timeout", !http.UseTimeout, "disable http timeout")
	rootCmd.PersistentFlags().Duration("timeout", http.TimeoutDuration, "max http request duration")

	rootCmd.PersistentFlags().Bool("no-read-env", false, "do not read enclave config from the environment")
	rootCmd.PersistentFlags().String("scope", ".", "the directory to scope your config to")
	rootCmd.PersistentFlags().String("configuration", configuration.UserConfigFile, "config file")
	rootCmd.PersistentFlags().Bool("json", false, "output json")
	rootCmd.PersistentFlags().Bool("debug", false, "output additional information when encountering errors")
}
