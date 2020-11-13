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

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/yaml.v3"
)

const repoConfigFileNameYaml = "doppler.yaml"
const repoConfigFileNameYml = "doppler.yml"

// RepoConfig Reads the configuration file (doppler.yaml) if exists and returns the set configuration
func RepoConfig() (models.RepoConfig, string, Error) {
	
	ymlFile := filepath.Join("./", repoConfigFileNameYml)
	yamlFile := filepath.Join("./", repoConfigFileNameYaml)
	var repoConfigFile string
	if utils.Exists(ymlFile) {
		repoConfigFile = ymlFile
	} else if utils.Exists(yamlFile) {
		repoConfigFile = yamlFile
	}
	if repoConfigFile != "" {
		utils.LogDebug(fmt.Sprintf("Reading repo config file %s", repoConfigFile))

		yamlFile, err := ioutil.ReadFile(repoConfigFile) // #nosec G304

		if err != nil {
			var e Error
			e.Err = err
			e.Message = "Unable to read doppler repo config file"
			return models.RepoConfig{}, repoConfigFile, e
		}

		var repoConfig models.RepoConfig

		if err := yaml.Unmarshal(yamlFile, &repoConfig); err != nil {
			var e Error
			e.Err = err
			e.Message = "Unable to parse doppler repo config file"
			return models.RepoConfig{}, repoConfigFile, e
		}

		return repoConfig, repoConfigFile, Error{}
	}
	return models.RepoConfig{}, repoConfigFile, Error{}
}
