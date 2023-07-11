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
	"sync"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/global"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var printConfig = false

var rootCmd = &cobra.Command{
	Use:   "doppler",
	Short: "The official Doppler CLI",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		loadFlags(cmd)
		configuration.Setup()
		configuration.LoadConfig()

		controllers.CaptureCommand(cmd.CommandPath())

		if utils.Debug && utils.Silent {
			utils.LogWarning("--silent has no effect when used with --debug")
		}

		// this output does not honor --silent
		if printConfig {
			utils.LogDebug("Active configuration")
			printer.ScopedConfigSource(configuration.LocalConfig(cmd), false, true, true)
			utils.LogDebug("")
		}

		plain := utils.GetBoolFlagIfChanged(cmd, "plain", false)
		canPrompt := !utils.GetBoolFlagIfChanged(cmd, "no-prompt", false) && !utils.GetBoolFlagIfChanged(cmd, "no-interactive", false)
		// tty is required to accept user input, otherwise the update can't be accepted/declined
		isTTY := isatty.IsTerminal(os.Stdout.Fd())

		// only run version check if we can print the results
		// --plain doesn't normally affect logging output, but due to legacy reasons it does here
		// also don't want to display updates if user doesn't want to be prompted (--no-prompt/--no-interactive)
		if isTTY && utils.CanLogInfo() && !plain && canPrompt {
			if available, latestVersion := controllers.CheckUpdate(cmd.CommandPath()); available {
				controllers.PromptToUpdate(latestVersion)
			}
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		err := cmd.Usage()
		if err != nil {
			utils.HandleError(err, "Unable to print command usage")
		}
	},
}

// persistentValidArgsFunction Cobra parses flags after executing ValidArgsFunction, so we must manually initialize flags
func persistentValidArgsFunction(cmd *cobra.Command) {
	// more info https://github.com/spf13/cobra/issues/1291
	loadFlags(cmd)
}

func loadFlags(cmd *cobra.Command) {
	var err error
	var normalizedScope string
	scope := cmd.Flag("scope").Value.String()
	if normalizedScope, err = configuration.NormalizeScope(scope); err != nil {
		utils.HandleError(err, fmt.Sprintf("Invalid scope: %s", scope))
	}
	configuration.Scope = normalizedScope

	configuration.CanReadEnv = !utils.GetBoolFlag(cmd, "no-read-env")

	// User Config Dir
	if configuration.CanReadEnv {
		userConfigDir := os.Getenv("DOPPLER_CONFIG_DIR")
		if userConfigDir != "" {
			utils.Log(valueFromEnvironmentNotice("DOPPLER_CONFIG_DIR"))
			configuration.SetConfigDir(userConfigDir)
		}
	}
	configuration.SetConfigDir(utils.GetPathFlagIfChanged(cmd, "config-dir", configuration.UserConfigDir))
	configuration.UserConfigFile = utils.GetPathFlagIfChanged(cmd, "configuration", configuration.UserConfigFile)
	http.UseTimeout = !utils.GetBoolFlag(cmd, "no-timeout")

	// DNS resolver
	if configuration.CanReadEnv {
		enableDNSResovler := os.Getenv("DOPPLER_ENABLE_DNS_RESOLVER")
		if enableDNSResovler == "true" {
			http.UseCustomDNSResolver = true
		} else if enableDNSResovler == "false" {
			http.UseCustomDNSResolver = false
		}
	}
	// flag takes precedence over env var
	http.UseCustomDNSResolver = utils.GetBoolFlagIfChanged(cmd, "enable-dns-resolver", http.UseCustomDNSResolver)

	// no-file is used by the 'secrets download' command to output secrets to stdout
	utils.Silent = utils.GetBoolFlagIfChanged(cmd, "no-file", utils.Silent)

	// version check
	if configuration.CanReadEnv {
		enable := os.Getenv("DOPPLER_ENABLE_VERSION_CHECK")
		if enable == "false" {
			utils.Log(valueFromEnvironmentNotice("DOPPLER_ENABLE_VERSION_CHECK"))
			version.PerformVersionCheck = false
		}
	}
	version.PerformVersionCheck = !utils.GetBoolFlagIfChanged(cmd, "no-check-version", !version.PerformVersionCheck)
}

func deprecatedCommand(newCommand string) {
	if newCommand == "" {
		utils.LogWarning("This command is deprecated")
	} else {
		utils.LogWarning(fmt.Sprintf("This command is deprecated, please use the \"doppler %s\" command", newCommand))
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// catch any panics in non-dev builds
	defer func() {
		if !version.IsDevelopment() {
			if err := recover(); err != nil {
				utils.Log(fmt.Sprintf("%s %v\n", color.Red.Render("Doppler Exception:"), err))
				os.Exit(1)
			}
		}
	}()

	// initialize the wait group before executing the command
	global.WaitGroup = new(sync.WaitGroup)

	err := rootCmd.Execute()

	// wait for group before checking error
	global.WaitGroup.Wait()

	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version.ProgramVersion
	rootCmd.SetVersionTemplate(rootCmd.Version + "\n")
	rootCmd.Flags().BoolP("version", "v", false, "Get the version of the Doppler CLI")

	rootCmd.PersistentFlags().StringP("token", "t", "", "doppler token")
	rootCmd.PersistentFlags().String("api-host", "https://api.doppler.com", "The host address for the Doppler API")
	rootCmd.PersistentFlags().String("dashboard-host", "https://dashboard.doppler.com", "The host address for the Doppler Dashboard")
	rootCmd.PersistentFlags().Bool("no-check-version", !version.PerformVersionCheck, "disable checking for Doppler CLI updates")
	rootCmd.PersistentFlags().Bool("no-verify-tls", false, "do not verify the validity of TLS certificates on HTTP requests (not recommended)")
	rootCmd.PersistentFlags().Bool("no-timeout", !http.UseTimeout, "disable http timeout")
	rootCmd.PersistentFlags().DurationVar(&http.TimeoutDuration, "timeout", http.TimeoutDuration, "max http request duration")
	rootCmd.PersistentFlags().IntVar(&http.RequestAttempts, "attempts", http.RequestAttempts, "number of http request attempts made before failing")
	// DNS resolver
	rootCmd.PersistentFlags().Bool("no-dns-resolver", !http.UseCustomDNSResolver, "use the OS's default DNS resolver")
	if err := rootCmd.PersistentFlags().MarkDeprecated("no-dns-resolver", "the DNS resolver is disabled by default"); err != nil {
		utils.HandleError(err)
	}
	if err := rootCmd.PersistentFlags().MarkHidden("no-dns-resolver"); err != nil {
		utils.HandleError(err)
	}
	rootCmd.PersistentFlags().Bool("enable-dns-resolver", http.UseCustomDNSResolver, "bypass the OS's default DNS resolver")
	rootCmd.PersistentFlags().StringVar(&http.DNSResolverAddress, "dns-resolver-address", http.DNSResolverAddress, "address to use for DNS resolution")
	rootCmd.PersistentFlags().StringVar(&http.DNSResolverProto, "dns-resolver-proto", http.DNSResolverProto, "protocol to use for DNS resolution")
	rootCmd.PersistentFlags().DurationVar(&http.DNSResolverTimeout, "dns-resolver-timeout", http.DNSResolverTimeout, "max dns lookup duration")

	rootCmd.PersistentFlags().Bool("no-read-env", false, "do not read config from the environment")
	rootCmd.PersistentFlags().String("scope", configuration.Scope, "the directory to scope your config to")
	rootCmd.PersistentFlags().String("config-dir", configuration.UserConfigDir, "config directory")
	rootCmd.PersistentFlags().String("configuration", configuration.UserConfigFile, "config file")
	if err := rootCmd.PersistentFlags().MarkDeprecated("configuration", "please use --config-dir instead"); err != nil {
		utils.HandleError(err)
	}
	if err := rootCmd.PersistentFlags().MarkHidden("configuration"); err != nil {
		utils.HandleError(err)
	}
	rootCmd.PersistentFlags().BoolVar(&utils.OutputJSON, "json", utils.OutputJSON, "output json")
	rootCmd.PersistentFlags().BoolVar(&utils.Debug, "debug", utils.Debug, "output additional information")
	rootCmd.PersistentFlags().BoolVar(&printConfig, "print-config", printConfig, "output active configuration")
	rootCmd.PersistentFlags().BoolVar(&utils.Silent, "silent", utils.Silent, "disable output of info messages")
}
