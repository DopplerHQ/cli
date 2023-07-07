/*
Copyright © 2020 Doppler <support@doppler.com>

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
	"github.com/DopplerHQ/cli/pkg/configuration"
	tuiApp "github.com/DopplerHQ/cli/pkg/tui"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch TUI (BETA)",
	Args:  cobra.NoArgs,
	Run:   tui,
}

func tui(cmd *cobra.Command, args []string) {
	localConfig := configuration.LocalConfig(cmd)
	tuiApp.Start(localConfig)
}

func init() {
	tuiCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	tuiCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	tuiCmd.Flags().BoolVar(&utils.DebugTUI, "debug-tui", utils.DebugTUI, "log TUI messages to file")
	rootCmd.AddCommand(tuiCmd)
}
