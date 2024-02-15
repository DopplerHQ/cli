/*
Copyright © 2023 Doppler <support@doppler.com>

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

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var configureFlagsCmd = &cobra.Command{
	Use:   "flags",
	Short: "View current flags",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		values := map[string]bool{}
		flags := models.GetFlags()
		for _, flag := range flags {
			value := configuration.GetFlag(flag)
			values[flag] = value
		}

		printer.Flags(values, utils.OutputJSON)
	},
}

var configureFlagsGetCmd = &cobra.Command{
	Use:               "get [flag]",
	Short:             "Get the value of a flag",
	ValidArgsFunction: FlagsValidArgs,
	Args:              cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		plain := utils.GetBoolFlag(cmd, "plain")

		flag := args[0]
		if !configuration.IsValidFlag(flag) {
			utils.HandleError(errors.New("invalid flag " + flag))
		}

		enabled := configuration.GetFlag(flag)

		printer.Flag(flag, enabled, utils.OutputJSON, plain, false)
	},
}

var configureFlagsEnableCmd = &cobra.Command{
	Use:               "enable [flag]",
	Short:             "Enable a flag",
	ValidArgsFunction: FlagsValidArgs,
	Args:              cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		flag := args[0]
		if !configuration.IsValidFlag(flag) {
			utils.HandleError(errors.New("invalid flag " + flag))
		}

		const value = true
		configuration.SetFlag(flag, value)

		if !utils.Silent {
			printer.Flag(flag, value, utils.OutputJSON, false, false)
		}
	},
}

var configureFlagsDisableCmd = &cobra.Command{
	Use:               "disable [flag]",
	Short:             "Disable a flag",
	ValidArgsFunction: FlagsValidArgs,
	Args:              cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		flag := args[0]
		if !configuration.IsValidFlag(flag) {
			utils.HandleError(errors.New("invalid flag " + flag))
		}

		const value = false
		configuration.SetFlag(flag, value)

		if !utils.Silent {
			printer.Flag(flag, value, utils.OutputJSON, false, false)
		}
	},
}

var configureFlagsResetCmd = &cobra.Command{
	Use:   "reset [flag]",
	Short: "Reset a flag to its default",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		flag := args[0]
		if !configuration.IsValidFlag(flag) {
			utils.HandleError(errors.New("invalid flag " + flag))
		}

		yes := utils.GetBoolFlag(cmd, "yes")
		defaultValue := configuration.GetFlagDefault(flag)

		if !yes {
			utils.PrintWarning(fmt.Sprintf("This will reset the %s flag to %t", flag, defaultValue))
			if !utils.ConfirmationPrompt("Continue?", false) {
				utils.Log("Aborting")
				return
			}
		}

		configuration.SetFlag(flag, defaultValue)
		printer.Flag(flag, defaultValue, utils.OutputJSON, false, false)
	},
}

func FlagsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	return models.GetFlags(), cobra.ShellCompDirectiveNoFileComp
}

func init() {
	configureCmd.AddCommand(configureFlagsCmd)

	configureFlagsGetCmd.Flags().Bool("plain", false, "print value without formatting")
	configureFlagsCmd.AddCommand(configureFlagsGetCmd)

	configureFlagsCmd.AddCommand(configureFlagsEnableCmd)

	configureFlagsCmd.AddCommand(configureFlagsDisableCmd)

	configureFlagsResetCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	configureFlagsCmd.AddCommand(configureFlagsResetCmd)

	rootCmd.AddCommand(configureFlagsCmd)
}
