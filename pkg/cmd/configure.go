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
	"errors"
	"sort"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "View the config file",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		all := utils.GetBoolFlag(cmd, "all")
		jsonFlag := utils.OutputJSON

		if all {
			printer.Configs(configuration.AllConfigs(), jsonFlag)
			return
		}

		scope := cmd.Flag("scope").Value.String()
		config := configuration.Get(scope)
		printer.ScopedConfig(config, jsonFlag)
	},
}

var configureDebugCmd = &cobra.Command{
	Use:   "debug",
	Short: "View current configuration utilizing all config sources",
	Long: `View current configuration utilizing all config sources.

This includes specified flags (--token=123), environment variables (DOPPLER_TOKEN=123),
and your config file. Flags have the highest priority; config file has the least.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON

		if !jsonFlag {
			color.Green.Printf("Configuration file: %s\n\n", configuration.UserConfigPath)
		}

		config := configuration.LocalConfig(cmd)
		printer.ScopedConfigSource(config, jsonFlag, true)
	},
}

var configureOptionsCmd = &cobra.Command{
	Use:   "options",
	Short: "List all supported config options",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON

		options := models.AllConfigOptions()
		sort.Strings(options)
		printer.ConfigOptionNames(options, jsonFlag)
	},
}

var configureGetCmd = &cobra.Command{
	Use:   "get [options]",
	Short: "Get the value of one or more options in the config file",
	Long: `Get the value of one or more options in the config file.

Ex: output the options "key" and "otherkey":
doppler configure get key otherkey`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("requires at least 1 arg(s), only received 0")
		}

		for _, arg := range args {
			if !configuration.IsValidConfigOption(arg) {
				return errors.New("invalid option " + arg)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		plain := utils.GetBoolFlag(cmd, "plain")

		scope := cmd.Flag("scope").Value.String()
		conf := configuration.Get(scope)

		printer.ScopedConfigValues(conf, args, models.ScopedPairs(&conf), jsonFlag, plain)
	},
}

var configureSetCmd = &cobra.Command{
	Use:   "set [options]",
	Short: "Set the value of one or more options in the config file",
	Long: `Set the value of one or more options in the config file.

Ex: set the options "key" and "otherkey":
doppler configure set key=123 otherkey=456`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("requires at least 1 arg(s), only received 0")
		}

		if !strings.Contains(args[0], "=") {
			if len(args) == 2 {
				if configuration.IsValidConfigOption(args[0]) {
					return nil
				}
				return errors.New("invalid option " + args[0])
			}

			return errors.New("too many arguments. To set multiple options, use the format option=value")
		}

		for _, arg := range args {
			option := strings.Split(arg, "=")
			if len(option) < 2 || !configuration.IsValidConfigOption(option[0]) {
				return errors.New("invalid option " + option[0])
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		silent := utils.GetBoolFlag(cmd, "silent")
		scope := cmd.Flag("scope").Value.String()
		jsonFlag := utils.OutputJSON

		if !strings.Contains(args[0], "=") {
			configuration.Set(scope, map[string]string{args[0]: args[1]})
		} else {
			options := map[string]string{}
			for _, option := range args {
				arr := strings.Split(option, "=")
				options[arr[0]] = arr[1]
			}
			configuration.Set(scope, options)
		}

		if !silent {
			printer.ScopedConfig(configuration.Get(scope), jsonFlag)
		}
	},
}

var configureUnsetCmd = &cobra.Command{
	Use:   "unset [options]",
	Short: "Unset the value of one or more options in the config file",
	Long: `Unset the value of one or more options in the config file.

Ex: unset the options "key" and "otherkey":
doppler configure unset key otherkey`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return errors.New("requires at least 1 arg(s), only received 0")
		}

		for _, arg := range args {
			if !configuration.IsValidConfigOption(arg) {
				return errors.New("invalid option " + arg)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		silent := utils.GetBoolFlag(cmd, "silent")
		jsonFlag := utils.OutputJSON

		scope := cmd.Flag("scope").Value.String()
		configuration.Unset(scope, args)

		if !silent {
			printer.ScopedConfig(configuration.Get(scope), jsonFlag)
		}
	},
}

func init() {
	configureCmd.AddCommand(configureDebugCmd)

	configureCmd.AddCommand(configureOptionsCmd)

	configureGetCmd.Flags().Bool("plain", false, "print values without formatting. values will be printed in the same order as specified")
	configureCmd.AddCommand(configureGetCmd)

	configureSetCmd.Flags().Bool("silent", false, "do not output the new config")
	configureCmd.AddCommand(configureSetCmd)

	configureUnsetCmd.Flags().Bool("silent", false, "do not output the new config")
	configureCmd.AddCommand(configureUnsetCmd)

	configureCmd.Flags().Bool("all", false, "print all saved options")
	rootCmd.AddCommand(configureCmd)
}
