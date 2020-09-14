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
	"fmt"
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

		config := configuration.Get(configuration.Scope)
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

		utils.Log(fmt.Sprintf("%s %s", color.Green.Render("Configuration file:"), configuration.UserConfigFile))

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
		copy := utils.GetBoolFlag(cmd, "copy")

		conf := configuration.Get(configuration.Scope)

		printer.ScopedConfigValues(conf, args, models.ScopedPairs(&conf), jsonFlag, plain, copy)
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
		jsonFlag := utils.OutputJSON

		if !strings.Contains(args[0], "=") {
			configuration.Set(configuration.Scope, map[string]string{args[0]: args[1]})
		} else {
			options := map[string]string{}
			for _, option := range args {
				arr := strings.Split(option, "=")
				options[arr[0]] = arr[1]
			}
			configuration.Set(configuration.Scope, options)
		}

		if !utils.Silent {
			printer.ScopedConfig(configuration.Get(configuration.Scope), jsonFlag)
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
		jsonFlag := utils.OutputJSON

		configuration.Unset(configuration.Scope, args)

		if !utils.Silent {
			printer.ScopedConfig(configuration.Get(configuration.Scope), jsonFlag)
		}
	},
}

func init() {
	configureCmd.AddCommand(configureDebugCmd)

	configureCmd.AddCommand(configureOptionsCmd)

	configureGetCmd.Flags().Bool("plain", false, "print values without formatting. values will be printed in the same order as specified")
	configureGetCmd.Flags().Bool("copy", false, "copy the value(s) to your clipboard")
	configureCmd.AddCommand(configureGetCmd)

	configureCmd.AddCommand(configureSetCmd)

	configureCmd.AddCommand(configureUnsetCmd)

	configureCmd.Flags().Bool("all", false, "print all saved options")
	rootCmd.AddCommand(configureCmd)
}
