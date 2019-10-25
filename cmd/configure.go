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
	configuration "doppler-cli/config"
	dopplerErrors "doppler-cli/errors"
	"doppler-cli/utils"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "View cli configuration",
	Run: func(cmd *cobra.Command, args []string) {
		all := utils.GetBoolFlag(cmd, "all")
		jsonFlag := utils.GetBoolFlag(cmd, "json")

		if all {
			allConfigs := configuration.AllConfigs()

			if jsonFlag {
				utils.PrintJSON(allConfigs)
				return
			}

			var rows [][]string
			for scope, config := range allConfigs {
				if config.Key != "" {
					rows = append(rows, []string{"key", config.Key, scope})
				}
				if config.Project != "" {
					rows = append(rows, []string{"project", config.Project, scope})
				}
				if config.Config != "" {
					rows = append(rows, []string{"config", config.Config, scope})
				}
			}

			utils.PrintTable([]string{"name", "value", "scope"}, rows)
			return
		}

		scope := cmd.Flag("scope").Value.String()
		config := configuration.Get(scope)

		if jsonFlag {
			confMap := make(map[string]map[string]string)

			if config.Config != (configuration.Pair{}) {
				scope := config.Config.Scope
				value := config.Config.Value

				if confMap[scope] == nil {
					confMap[scope] = make(map[string]string)
				}
				confMap[scope]["config"] = value
			}

			if config.Project != (configuration.Pair{}) {
				scope := config.Project.Scope
				value := config.Project.Value

				if confMap[scope] == nil {
					confMap[scope] = make(map[string]string)
				}
				confMap[scope]["project"] = value
			}

			if config.Key != (configuration.Pair{}) {
				scope := config.Key.Scope
				value := config.Key.Value

				if confMap[scope] == nil {
					confMap[scope] = make(map[string]string)
				}
				confMap[scope]["key"] = value
			}

			utils.PrintJSON(confMap)
			return
		}

		rows := [][]string{{"key", config.Key.Value, config.Key.Scope}, {"project", config.Project.Value, config.Project.Scope}, {"config", config.Config.Value, config.Config.Scope}}
		utils.PrintTable([]string{"name", "value", "scope"}, rows)
	},
}

var configureGetCmd = &cobra.Command{
	Use:   "get [options]",
	Short: "Get the value of one or more config options",
	Long: `Get the value of one or more config options.

Ex: output the options "key" and "otherkey":
doppler configure get key otherkey`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		jsonFlag := utils.GetBoolFlag(cmd, "json")
		plain := utils.GetBoolFlag(cmd, "plain")

		scope := cmd.Flag("scope").Value.String()
		conf := configuration.Get(scope)

		if plain {
			sbEmpty := true
			var sb strings.Builder

			for _, arg := range args {
				value := ""
				if arg == "key" {
					value = conf.Key.Value
				} else if arg == "project" {
					value = conf.Project.Value
				} else if arg == "config" {
					value = conf.Config.Value
				}

				if value != "" {
					if sbEmpty {
						sbEmpty = false
					} else {
						sb.WriteString("\n")
					}

					sb.WriteString(value)
				}
			}

			fmt.Println(sb.String())
			return
		}

		if jsonFlag {
			filteredConfMap := make(map[string]string)
			for _, arg := range args {
				if arg == "key" {
					filteredConfMap["key"] = conf.Key.Value
				} else if arg == "project" {
					filteredConfMap["project"] = conf.Project.Value
				} else if arg == "config" {
					filteredConfMap["config"] = conf.Config.Value
				}
			}

			utils.PrintJSON(filteredConfMap)
			return
		}

		var rows [][]string
		for _, arg := range args {
			if arg == "key" {
				rows = append(rows, []string{"key", conf.Key.Value, conf.Key.Scope})
			} else if arg == "project" {
				rows = append(rows, []string{"project", conf.Project.Value, conf.Project.Scope})
			} else if arg == "config" {
				rows = append(rows, []string{"config", conf.Config.Value, conf.Config.Scope})
			}
		}

		utils.PrintTable([]string{"name", "value", "scope"}, rows)
	},
}

var configureSetCmd = &cobra.Command{
	Use:   "set [options]",
	Short: "Set the value of one or more config options",
	Long: `Set the value of one or more config options.

Ex: set the options "key" and "otherkey":
doppler configure set key=123 otherkey=456`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		silent := utils.GetBoolFlag(cmd, "silent")

		scope := cmd.Flag("scope").Value.String()
		configuration.Set(scope, args)

		if !silent {
			conf := configuration.Get(scope)
			rows := [][]string{{"key", conf.Key.Value, conf.Key.Scope}, {"project", conf.Project.Value, conf.Project.Scope}, {"config", conf.Config.Value, conf.Config.Scope}}
			utils.PrintTable([]string{"name", "value", "scope"}, rows)
		}
	},
}

var configureUnsetCmd = &cobra.Command{
	Use:   "unset [options]",
	Short: "Unset the value of one or more config options",
	Long: `Unset the value of one or more config options.

Ex: unset the options "key" and "otherkey":
doppler configure unset key otherkey`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			dopplerErrors.CommandMissingArgument(cmd)
		}

		silent := utils.GetBoolFlag(cmd, "silent")

		scope := cmd.Flag("scope").Value.String()
		configuration.Unset(scope, args)

		if !silent {
			conf := configuration.Get(scope)
			rows := [][]string{{"key", conf.Key.Value, conf.Key.Scope}, {"project", conf.Project.Value, conf.Project.Scope}, {"config", conf.Config.Value, conf.Config.Scope}}
			utils.PrintTable([]string{"name", "value", "scope"}, rows)
		}
	},
}

func init() {
	configureGetCmd.Flags().Bool("plain", false, "print values without formatting. values will be printed in the same order as specified	")
	configureGetCmd.Flags().Bool("json", false, "output json")
	configureCmd.AddCommand(configureGetCmd)

	configureSetCmd.Flags().Bool("silent", false, "don't output the new config")
	configureCmd.AddCommand(configureSetCmd)

	configureUnsetCmd.Flags().Bool("silent", false, "don't output the new config")
	configureCmd.AddCommand(configureUnsetCmd)

	configureCmd.Flags().Bool("all", false, "print all saved options")
	configureCmd.Flags().Bool("json", false, "output json")
	rootCmd.AddCommand(configureCmd)
}
