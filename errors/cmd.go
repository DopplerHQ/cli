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
package errors

import (
	"os"

	"github.com/DopplerHQ/cli/utils"
	"github.com/spf13/cobra"
)

// CommandMissingArgument command missing an argument
func CommandMissingArgument(cmd *cobra.Command) {
	utils.Log("Error: command needs an argument")
	cmd.Help()
	os.Exit(1)
}

// CommandMissingSubcommand command missing a subcommand
func CommandMissingSubcommand(cmd *cobra.Command) {
	cmd.Help()
	os.Exit(1)
}

// CommandMissingFlag command missing a flag
func CommandMissingFlag(cmd *cobra.Command) {
	utils.Log("Error: command needs a flag")
	cmd.Help()
	os.Exit(1)
}

// ApplicationMissingCommand application missing a command
func ApplicationMissingCommand(cmd *cobra.Command) {
	utils.Log("Error: application needs a command")
	cmd.Help()
	os.Exit(1)
}
