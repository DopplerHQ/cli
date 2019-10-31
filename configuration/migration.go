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
package configuration

import (
	"encoding/json"
	"io/ioutil"

	"github.com/DopplerHQ/cli/models"
	"github.com/DopplerHQ/cli/utils"
)

type oldConfig struct {
	Pipeline    string
	Environment string
	Key         string
}

var jsonFile = utils.Home() + "/.doppler.json"

func jsonExists() bool {
	return utils.Exists(jsonFile)
}

// migrateJSONToYaml migrate ~/.doppler.json to  ~/.doppler.yaml
func migrateJSONToYaml() {
	jsonConfig := parseJSONConfig()
	newConfig := convertOldConfig(jsonConfig)
	writeYAML(newConfig)
}

func convertOldConfig(oldConfig map[string]oldConfig) map[string]models.Config {
	config := make(map[string]models.Config)

	for key, val := range oldConfig {
		config[key] = models.Config{Project: val.Pipeline, Config: val.Environment, Key: val.Key}
	}

	return config
}

func parseJSONConfig() map[string]oldConfig {
	fileContents, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		utils.Err(err, "")
	}

	var config map[string]oldConfig
	err = json.Unmarshal(fileContents, &config)
	if err != nil {
		utils.Err(err, "")
	}

	return config
}
