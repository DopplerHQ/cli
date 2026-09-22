/*
Copyright © 2026 Doppler <support@doppler.com>

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
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

var validTagColors = []string{"gray", "purple", "blue", "green", "yellow", "orange", "red", "pink"}
var validTagColorsList = strings.Join(validTagColors, ", ")

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage workplace tags",
	Args:  cobra.NoArgs,
	Run:   tags,
}

var tagsGetCmd = &cobra.Command{
	Use:               "get [slug]",
	Short:             "Get info for a tag",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: tagSlugsValidArgs,
	Run:               getTag,
}

var tagsCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a tag",
	Args:  cobra.ExactArgs(1),
	Run:   createTag,
}

var tagsUpdateCmd = &cobra.Command{
	Use:   "update [slug]",
	Short: "Update a tag",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			return err
		}

		if cmd.Flag("name").Value.String() == "" && cmd.Flag("color").Value.String() == "" {
			return errors.New("command needs flag --name or --color")
		}

		return nil
	},
	ValidArgsFunction: tagSlugsValidArgs,
	Run:               updateTag,
}

var tagsDeleteCmd = &cobra.Command{
	Use:               "delete [slug]",
	Short:             "Delete a tag",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: tagSlugsValidArgs,
	Run:               deleteTag,
}

func tags(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	utils.RequireValue("token", localConfig.Token.Value)

	info, err := http.GetTags(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.TagsInfo(info, jsonFlag)
}

func getTag(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	localConfig := configuration.LocalConfig(cmd)

	slug := args[0]

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("slug", slug)

	info, err := http.GetTag(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, slug)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	printer.TagInfo(info, jsonFlag)
}

func createTag(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	color := cmd.Flag("color").Value.String()
	slug := cmd.Flag("slug").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	name := args[0]

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("name", name)

	info, err := http.CreateTag(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, name, color, slug)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.TagInfo(info, jsonFlag)
	}
}

func updateTag(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	name := cmd.Flag("name").Value.String()
	color := cmd.Flag("color").Value.String()
	localConfig := configuration.LocalConfig(cmd)

	slug := args[0]

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("slug", slug)

	info, err := http.UpdateTag(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, slug, name, color)
	if !err.IsNil() {
		utils.HandleError(err.Unwrap(), err.Message)
	}

	if !utils.Silent {
		printer.TagInfo(info, jsonFlag)
	}
}

func deleteTag(cmd *cobra.Command, args []string) {
	jsonFlag := utils.OutputJSON
	yes := utils.GetBoolFlag(cmd, "yes")
	localConfig := configuration.LocalConfig(cmd)

	slug := args[0]

	utils.RequireValue("token", localConfig.Token.Value)
	utils.RequireValue("slug", slug)

	if yes || utils.ConfirmationPrompt(fmt.Sprintf("Delete tag %s", slug), false) {
		err := http.DeleteTag(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, slug)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}

		if !utils.Silent {
			info, err := http.GetTags(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}

			printer.TagsInfo(info, jsonFlag)
		}
	}
}

func tagSlugsValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	persistentValidArgsFunction(cmd)

	localConfig := configuration.LocalConfig(cmd)
	slugs, err := controllers.GetTagSlugs(localConfig)
	if err.IsNil() {
		return slugs, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	tagsCmd.AddCommand(tagsGetCmd)

	tagsCreateCmd.Flags().String("color", "", fmt.Sprintf("tag color. One of: %s", validTagColorsList))
	tagsCreateCmd.Flags().String("slug", "", "tag slug (generated from the name when omitted)")
	tagsCmd.AddCommand(tagsCreateCmd)

	tagsUpdateCmd.Flags().String("name", "", "new name")
	tagsUpdateCmd.Flags().String("color", "", fmt.Sprintf("new color. One of: %s", validTagColorsList))
	tagsCmd.AddCommand(tagsUpdateCmd)

	tagsDeleteCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation")
	tagsCmd.AddCommand(tagsDeleteCmd)

	rootCmd.AddCommand(tagsCmd)
}
