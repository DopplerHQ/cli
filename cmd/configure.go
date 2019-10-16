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
	"doppler-cli/utils"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// configureCmd represents the config command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "View cli configuration",
	Run: func(cmd *cobra.Command, args []string) {
		all, err := strconv.ParseBool(cmd.Flag("all").Value.String())
		if err != nil {
			utils.Err(err)
		}

		jsonFlag, err := strconv.ParseBool(cmd.Flag("json").Value.String())
		if err != nil {
			utils.Err(err)
		}

		if all {
			allConfigs := configuration.AllConfigs()

			if jsonFlag {
				resp, err := json.Marshal(allConfigs)
				if err != nil {
					utils.Err(err)
				}

				fmt.Println(string(resp))
				return
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"name", "value", "scope"})

			for scope, config := range allConfigs {
				if config.Key != "" {
					table.Append([]string{"key", config.Key, scope})
				}
				if config.Project != "" {
					table.Append([]string{"project", config.Project, scope})
				}
				if config.Config != "" {
					table.Append([]string{"config", config.Config, scope})
				}
			}

			table.Render()
			return
		}

		scope := cmd.Flag("scope").Value.String()

		if jsonFlag {
			resp, err := json.Marshal(configuration.Get(scope))
			if err != nil {
				utils.Err(err)
			}

			fmt.Println(string(resp))
			return
		}

		config := configuration.Get(scope)
		var data [][]string
		data = append(data, []string{"key", config.Key.Value, config.Key.Scope})
		data = append(data, []string{"project", config.Project.Value, config.Project.Scope})
		data = append(data, []string{"config", config.Config.Value, config.Config.Scope})

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"name", "value", "scope"})

		for _, row := range data {
			table.Append(row)
		}

		table.Render()
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
			fmt.Println("Error: missing argument")
			cmd.Help()
			return
		}

		plain, err := strconv.ParseBool(cmd.Flag("plain").Value.String())
		if err != nil {
			utils.Err(err)
		}

		jsonFlag, err := strconv.ParseBool(cmd.Flag("json").Value.String())
		if err != nil {
			utils.Err(err)
		}

		scope := cmd.Flag("scope").Value.String()
		conf := configuration.Get(scope)

		if plain {
			sbEmpty := true
			var sb strings.Builder

			for _, arg := range args {
				if arg == "key" {
					if sbEmpty {
						sbEmpty = false
					} else {
						sb.WriteString("\n")
					}
					sb.WriteString(conf.Key.Value)
				}
				if arg == "project" {
					if sbEmpty {
						sbEmpty = false
					} else {
						sb.WriteString("\n")
					}
					sb.WriteString(conf.Project.Value)
				}
				if arg == "config" {
					if sbEmpty {
						sbEmpty = false
					} else {
						sb.WriteString("\n")
					}
					sb.WriteString(conf.Config.Value)
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
				}
				if arg == "project" {
					filteredConfMap["project"] = conf.Project.Value
				}
				if arg == "config" {
					filteredConfMap["config"] = conf.Config.Value
				}
			}

			resp, err := json.Marshal(filteredConfMap)
			if err != nil {
				utils.Err(err)
			}

			fmt.Println(string(resp))
			return
		}

		var data [][]string
		for _, arg := range args {
			if arg == "key" {
				data = append(data, []string{"key", conf.Key.Value, conf.Key.Scope})
			}
			if arg == "project" {
				data = append(data, []string{"project", conf.Project.Value, conf.Project.Scope})
			}
			if arg == "config" {
				data = append(data, []string{"config", conf.Config.Value, conf.Config.Scope})
			}
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"name", "value", "scope"})

		for _, row := range data {
			table.Append(row)
		}

		table.Render()
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
			fmt.Println("Error: command needs an argument")
			cmd.Help()
			return
		}

		silent, err := strconv.ParseBool(cmd.Flag("silent").Value.String())
		if err != nil {
			utils.Err(err)
		}

		scope := cmd.Flag("scope").Value.String()
		configuration.Set(scope, args)

		if !silent {
			conf := configuration.Get(scope)
			var data [][]string
			data = append(data, []string{"key", conf.Key.Value, conf.Key.Scope})
			data = append(data, []string{"project", conf.Project.Value, conf.Project.Scope})
			data = append(data, []string{"config", conf.Config.Value, conf.Config.Scope})

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"name", "value", "scope"})

			for _, row := range data {
				table.Append(row)
			}

			table.Render()
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
			fmt.Println("Error: command needs an argument")
			cmd.Help()
			return
		}

		silent, err := strconv.ParseBool(cmd.Flag("silent").Value.String())
		if err != nil {
			utils.Err(err)
		}

		scope := cmd.Flag("scope").Value.String()
		configuration.Unset(scope, args)

		if !silent {
			conf := configuration.Get(scope)
			var data [][]string
			data = append(data, []string{"key", conf.Key.Value, conf.Key.Scope})
			data = append(data, []string{"project", conf.Project.Value, conf.Project.Scope})
			data = append(data, []string{"config", conf.Config.Value, conf.Config.Scope})

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"name", "value", "scope"})

			for _, row := range data {
				table.Append(row)
			}

			table.Render()
		}
	},
}

func init() {
	configureCmd.Flags().Bool("all", false, "print all saved options")

	configureGetCmd.Flags().Bool("plain", false, "print values without formatting. values will be printed in the same order as specified	")

	configureSetCmd.Flags().Bool("silent", false, "don't output the new config")

	configureUnsetCmd.Flags().Bool("silent", false, "don't output the new config")

	configureCmd.AddCommand(configureGetCmd)
	configureCmd.AddCommand(configureSetCmd)
	configureCmd.AddCommand(configureUnsetCmd)
	rootCmd.AddCommand(configureCmd)
}
