/*
Copyright © 2021 Doppler <support@doppler.com>

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
	"io/ioutil"
	"path/filepath"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/printer"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
)

const projectTemplateFileName = "doppler-template.yaml"

var importCommand = &cobra.Command{
	Use:   "import",
	Short: "Import projects into your Doppler workplace",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag := utils.OutputJSON
		localConfig := configuration.LocalConfig(cmd)
		utils.RequireValue("token", localConfig.Token.Value)
		projectTemplateFile, err := utils.GetFilePath(cmd.Flag("template").Value.String())
		if err != nil {
			utils.HandleError(err, "Unable to parse template file path")
		}
		template := readTemplateFile(projectTemplateFile)

		info, importErr := http.ImportTemplate(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, template)
		if !importErr.IsNil() {
			utils.HandleError(importErr.Unwrap(), importErr.Message)
		}
		printer.ProjectsInfo(info, jsonFlag)
	},
}

func readTemplateFile(projectTemplateFile string) []byte {
	if !utils.Exists(projectTemplateFile) {
		utils.HandleError(fmt.Errorf("Unable to find project template file: %s", projectTemplateFile))
	}
	utils.LogDebug(fmt.Sprintf("Reading template file %s", projectTemplateFile))
	yamlFile, err := ioutil.ReadFile(projectTemplateFile) // #nosec G304
	if err != nil {
		utils.HandleError(err, fmt.Sprintf("Unable to read project template file: %s", projectTemplateFile))
	}
	return yamlFile
}

func init() {
	importCommand.Flags().String("template", filepath.Join("./", projectTemplateFileName), "path to template file (e.g. './path/to/file.yaml')")
	rootCmd.AddCommand(importCommand)
}
