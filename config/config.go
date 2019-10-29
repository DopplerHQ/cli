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
package config

import (
	"doppler-cli/models"
	utils "doppler-cli/utils"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var yamlFile = utils.Home() + "/.doppler.yaml"

var configContents map[string]models.Config

func init() {
	if !exists() {
		if jsonExists() {
			migrateJSONToYaml()
		} else {
			var blankConfig map[string]models.Config
			writeYAML(blankConfig)
		}
	}

	configContents = readYAML()
}

// Get the config at the specified scope
func Get(scope string) models.ScopedConfig {
	scope, err := parseScope(scope)
	if err != nil {
		utils.Err(err, "")
	}
	scope = path.Join(scope, "/")
	var scopedConfig models.ScopedConfig

	for confScope, conf := range configContents {
		// both paths should end in / to prevent martial match (e.g. /test matching /test123)
		if confScope != "*" && !strings.HasPrefix(scope, path.Join(confScope, "/")) {
			continue
		}

		if conf.Key != "" {
			if scopedConfig.Key == (models.Pair{}) || len(confScope) > len(scopedConfig.Key.Scope) {
				scopedConfig.Key.Value = conf.Key
				scopedConfig.Key.Scope = confScope
			}
		}

		if conf.Project != "" {
			if scopedConfig.Project == (models.Pair{}) || len(confScope) > len(scopedConfig.Project.Scope) {
				scopedConfig.Project.Value = conf.Project
				scopedConfig.Project.Scope = confScope
			}
		}

		if conf.Config != "" {
			if scopedConfig.Config == (models.Pair{}) || len(confScope) > len(scopedConfig.Config.Scope) {
				scopedConfig.Config.Value = conf.Config
				scopedConfig.Config.Scope = confScope
			}
		}

		if conf.APIHost != "" {
			if scopedConfig.APIHost == (models.Pair{}) || len(confScope) > len(scopedConfig.APIHost.Scope) {
				scopedConfig.APIHost.Value = conf.APIHost
				scopedConfig.APIHost.Scope = confScope
			}
		}

		if conf.DeployHost != "" {
			if scopedConfig.DeployHost == (models.Pair{}) || len(confScope) > len(scopedConfig.DeployHost.Scope) {
				scopedConfig.DeployHost.Value = conf.DeployHost
				scopedConfig.DeployHost.Scope = confScope
			}
		}
	}

	return scopedConfig
}

// LocalConfig retrieves the config for the scoped directory
func LocalConfig(cmd *cobra.Command) models.ScopedConfig {
	// cli config file (lowest priority)
	localConfig := Get(cmd.Flag("scope").Value.String())

	// environment variables
	if !utils.GetBoolFlag(cmd, "no-read-env") {
		key := os.Getenv("DOPPLER_API_KEY")
		if key != "" {
			localConfig.Key.Value = key
			localConfig.Key.Scope = ""
		}

		project := os.Getenv("DOPPLER_PROJECT")
		if project != "" {
			localConfig.Project.Value = project
			localConfig.Project.Scope = ""
		}

		config := os.Getenv("DOPPLER_CONFIG")
		if config != "" {
			localConfig.Config.Value = config
			localConfig.Config.Scope = ""
		}

		apiHost := os.Getenv("DOPPLER_API_HOST")
		if apiHost != "" {
			localConfig.APIHost.Value = apiHost
			localConfig.APIHost.Scope = ""
		}

		deployHost := os.Getenv("DOPPLER_DEPLOY_HOST")
		if deployHost != "" {
			localConfig.DeployHost.Value = deployHost
			localConfig.DeployHost.Scope = ""
		}
	}

	// individual flags (highest priority)
	if cmd.Flags().Changed("key") || localConfig.Key.Value == "" {
		localConfig.Key.Value = cmd.Flag("key").Value.String()
		localConfig.Key.Scope = ""
	}

	if cmd.Flags().Changed("project") || localConfig.Project.Value == "" {
		localConfig.Project.Value = cmd.Flag("project").Value.String()
		localConfig.Project.Scope = ""
	}

	if cmd.Flags().Changed("config") || localConfig.Config.Value == "" {
		localConfig.Config.Value = cmd.Flag("config").Value.String()
		localConfig.Config.Scope = ""
	}

	if cmd.Flags().Changed("api-host") || localConfig.APIHost.Value == "" {
		localConfig.APIHost.Value = cmd.Flag("api-host").Value.String()
		localConfig.APIHost.Scope = ""
	}

	if cmd.Flags().Changed("deploy-host") || localConfig.DeployHost.Value == "" {
		localConfig.DeployHost.Value = cmd.Flag("deploy-host").Value.String()
		localConfig.DeployHost.Scope = ""
	}

	return localConfig
}

// AllConfigs get all configs we know about
func AllConfigs() map[string]models.Config {
	return configContents
}

// Set a local config
func Set(scope string, options map[string]string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.Err(err, "")
		}
	}

	for key, value := range options {
		if !IsValidConfigOption(key) {
			utils.Err(errors.New("invalid option "+key), "")
		}

		scopedConfig := configContents[scope]
		if key == "key" {
			scopedConfig.Key = value
		} else if key == "project" {
			scopedConfig.Project = value
		} else if key == "config" {
			scopedConfig.Config = value
		} else if key == "api-host" {
			scopedConfig.APIHost = value
		} else if key == "deploy-host" {
			scopedConfig.DeployHost = value
		}
		configContents[scope] = scopedConfig
	}

	writeYAML(configContents)
}

// Unset a local config
func Unset(scope string, options []string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.Err(err, "")
		}
	}

	if configContents[scope] == (models.Config{}) {
		return
	}

	for _, key := range options {
		if !IsValidConfigOption(key) {
			utils.Err(errors.New("invalid option "+key), "")
		}

		scopedConfig := configContents[scope]
		if key == "key" {
			scopedConfig.Key = ""
		} else if key == "project" {
			scopedConfig.Project = ""
		} else if key == "config" {
			scopedConfig.Config = ""
		} else if key == "api-host" {
			scopedConfig.APIHost = ""
		} else if key == "deploy-host" {
			scopedConfig.DeployHost = ""
		}
		configContents[scope] = scopedConfig
	}

	if configContents[scope] == (models.Config{}) {
		delete(configContents, scope)
	}

	writeYAML(configContents)
}

// Write config to ~/.doppler.yaml
func writeYAML(config map[string]models.Config) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		utils.Err(err, "")
	}

	err = ioutil.WriteFile(yamlFile, bytes, os.FileMode(0600))
	if err != nil {
		utils.Err(err, "")
	}
}

func exists() bool {
	return utils.Exists(yamlFile)
}

func readYAML() map[string]models.Config {
	fileContents, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		utils.Err(err, "")
	}

	var config map[string]models.Config
	yaml.Unmarshal(fileContents, &config)
	return config
}

func parseScope(scope string) (string, error) {
	absScope, err := filepath.Abs(scope)
	if err != nil {
		return "", err
	}

	return absScope, nil
}

// IsValidConfigOption whether the specified key is a valid option
func IsValidConfigOption(key string) bool {
	return key == "key" || key == "project" || key == "config" || key == "api-host" || key == "deploy-host"
}

// GetConfigValue get the value of the specified key within the config
func GetConfigValue(conf models.ScopedConfig, key string) (string, string) {
	if key == "key" {
		return conf.Key.Value, conf.Key.Scope
	}
	if key == "project" {
		return conf.Project.Value, conf.Project.Scope
	}
	if key == "config" {
		return conf.Config.Value, conf.Config.Scope
	}
	if key == "api-host" {
		return conf.APIHost.Value, conf.APIHost.Scope
	}
	if key == "deploy-host" {
		return conf.DeployHost.Value, conf.DeployHost.Scope
	}

	return "", ""
}
