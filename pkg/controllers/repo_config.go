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
package controllers

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

// repoConfigFileName (doppler.yaml)
const repoConfigFileName = "doppler.yaml"

// ymlRepoConfigFileName (doppler.yml)
const ymlRepoConfigFileName = "doppler.yml"

// RepoConfig Reads the configuration file (doppler.yaml) if exists and returns the set configuration
func RepoConfig() (models.MultiRepoConfig, Error) {

	repoConfigFile := filepath.Join("./", repoConfigFileName)
	ymlRepoConfigFile := filepath.Join("./", ymlRepoConfigFileName)

	if utils.Exists(repoConfigFile) {
		utils.LogDebug(fmt.Sprintf("Reading repo config file %s", repoConfigFile))

		yamlFile, err := ioutil.ReadFile(repoConfigFile) // #nosec G304

		if err != nil {
			var e Error
			e.Err = err
			e.Message = "Unable to read doppler repo config file"
			return models.MultiRepoConfig{}, e
		}

		var repoConfig models.MultiRepoConfig

		if err := yaml.Unmarshal(yamlFile, &repoConfig); err != nil {
			// Try parsing old repoConfig format (i.e., no slice) for backwards compatibility
			var oldRepoConfig models.RepoConfig
			if err := yaml.Unmarshal(yamlFile, &oldRepoConfig); err != nil {
				var e Error
				e.Err = err
				e.Message = "Unable to parse doppler repo config file"
				return models.MultiRepoConfig{}, e
			} else {
				repoConfig.Setup = append(repoConfig.Setup, oldRepoConfig.Setup)
				return repoConfig, Error{}
			}
		}

		return repoConfig, Error{}
	} else if utils.Exists(ymlRepoConfigFile) {
		utils.LogWarning(fmt.Sprintf("Found %s file, please rename to %s for repo configuration", ymlRepoConfigFile, repoConfigFileName))
	} else {
		// If no config file exists, then this is for an interactive setup, so
		// return a MultiRepoConfig object containing an empty ProjectConfig object
		var repoConfig models.MultiRepoConfig
		repoConfig.Setup = append(repoConfig.Setup, models.ProjectConfig{Path: configuration.Scope})
		return repoConfig, Error{}
	}
	return models.MultiRepoConfig{}, Error{}
}
