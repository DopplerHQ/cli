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
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var enclaveConfigsCmd = &cobra.Command{
	Use:   "configs",
	Short: "List Enclave configs",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs")
		configs(cmd, args)
	},
}

var enclaveConfigsGetCmd = &cobra.Command{
	Use:   "get [config]",
	Short: "Get info for a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs get")
		getConfigs(cmd, args)
	},
}

var enclaveConfigsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs create")
		createConfigs(cmd, args)
	},
}

var enclaveConfigsDeleteCmd = &cobra.Command{
	Use:   "delete [config]",
	Short: "Delete a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs delete")
		deleteConfigs(cmd, args)
	},
}

var enclaveConfigsUpdateCmd = &cobra.Command{
	Use:   "update [config]",
	Short: "Update a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs update")
		updateConfigs(cmd, args)
	},
}

var enclaveConfigsLockCmd = &cobra.Command{
	Use:   "lock [config]",
	Short: "Lock a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs lock")
		lockConfigs(cmd, args)
	},
}

var enclaveConfigsUnlockCmd = &cobra.Command{
	Use:   "unlock [config]",
	Short: "Unlock a config",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		deprecatedCommand("configs unlock")
		unlockConfigs(cmd, args)
	},
}

func init() {
	enclaveConfigsCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCmd.Flags().StringP("environment", "e", "", "config environment")
	if err := enclaveConfigsCmd.RegisterFlagCompletionFunc("environment", configEnvironmentIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCmd.Flags().IntP("number", "n", 100, "max number of configs to display")
	enclaveConfigsCmd.Flags().Int("page", 1, "page to display")

	enclaveConfigsGetCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsGetCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsGetCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsGetCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCmd.AddCommand(enclaveConfigsGetCmd)

	enclaveConfigsCreateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsCreateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCreateCmd.Flags().String("name", "", "config name")
	enclaveConfigsCreateCmd.Flags().StringP("environment", "e", "", "config environment")
	if err := enclaveConfigsCreateCmd.RegisterFlagCompletionFunc("environment", configEnvironmentIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsCmd.AddCommand(enclaveConfigsCreateCmd)

	enclaveConfigsUpdateCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsUpdateCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsUpdateCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsUpdateCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsUpdateCmd.Flags().String("name", "", "config name")
	if err := enclaveConfigsUpdateCmd.MarkFlagRequired("name"); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsUpdateCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveConfigsCmd.AddCommand(enclaveConfigsUpdateCmd)

	enclaveConfigsDeleteCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsDeleteCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsDeleteCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsDeleteCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveConfigsCmd.AddCommand(enclaveConfigsDeleteCmd)

	enclaveConfigsLockCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsLockCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLockCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsLockCmd.RegisterFlagCompletionFunc("config", lockedConfigNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsLockCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveConfigsCmd.AddCommand(enclaveConfigsLockCmd)

	enclaveConfigsUnlockCmd.Flags().StringP("project", "p", "", "enclave project (e.g. backend)")
	if err := enclaveConfigsUnlockCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsUnlockCmd.Flags().StringP("config", "c", "", "enclave config (e.g. dev)")
	if err := enclaveConfigsUnlockCmd.RegisterFlagCompletionFunc("config", unlockedConfigNamesValidArgs); err != nil {
		utils.HandleError(err)
	}
	enclaveConfigsUnlockCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	enclaveConfigsCmd.AddCommand(enclaveConfigsUnlockCmd)

	enclaveCmd.AddCommand(enclaveConfigsCmd)
}
