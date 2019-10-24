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

// Config options
type Config struct {
	Project string `json:"project"`
	Config  string `json:"config"`
	Key     string `json:"key"`
}

// ScopedConfig options with their scope
type ScopedConfig struct {
	Project Pair `json:"project"`
	Config  Pair `json:"config"`
	Key     Pair `json:"key"`
}

type Pair struct {
	Value string `json:"value"`
	Scope string `json:"scope"`
}

var yamlFile = utils.Home() + "/.doppler.yaml"

var configContents map[string]Config

func init() {
	if !exists() {
		if jsonExists() {
			migrateJSONToYaml()
		} else {
			var blankConfig map[string]Config
			writeYAML(blankConfig)
		}
	}

	configContents = readYAML()
}

// Get the config at the specified scope
func Get(scope string) ScopedConfig {
	scope, err := parseScope(scope)
	if err != nil {
		utils.Err(err)
	}
	scope = path.Join(scope, "/")
	var scopedConfig ScopedConfig

	for confScope, conf := range configContents {
		// both paths should end in / to prevent martial match (e.g. /test matching /test123)
		if confScope != "*" && !strings.HasPrefix(scope, path.Join(confScope, "/")) {
			continue
		}

		if conf.Key != "" {
			if scopedConfig.Key == (Pair{}) {
				scopedConfig.Key = Pair{Value: conf.Key, Scope: confScope}
			} else if len(confScope) > len(scopedConfig.Key.Scope) {
				scopedConfig.Key.Value = conf.Key
				scopedConfig.Key.Scope = confScope
			}
		}

		if conf.Project != "" {
			if scopedConfig.Project == (Pair{}) {
				scopedConfig.Project = Pair{Value: conf.Project, Scope: confScope}
			} else if len(confScope) > len(scopedConfig.Project.Scope) {
				scopedConfig.Project.Value = conf.Project
				scopedConfig.Project.Scope = confScope
			}
		}

		if conf.Config != "" {
			if scopedConfig.Config == (Pair{}) {
				scopedConfig.Config = Pair{Value: conf.Config, Scope: confScope}
			} else if len(confScope) > len(scopedConfig.Config.Scope) {
				scopedConfig.Config.Value = conf.Config
				scopedConfig.Config.Scope = confScope
			}
		}
	}

	return scopedConfig
}

// LocalConfig retrieves the config for the current directory. prioritizes command flags and falls back to the config file
func LocalConfig(cmd *cobra.Command) ScopedConfig {
	localConfig := Get(cmd.Flag("scope").Value.String())

	if cmd.Flags().Changed("key") {
		localConfig.Key.Value = cmd.Flag("key").Value.String()
		localConfig.Key.Scope = ""
	}

	if cmd.Flags().Changed("project") {
		localConfig.Project.Value = cmd.Flag("project").Value.String()
		localConfig.Project.Scope = ""
	}

	if cmd.Flags().Changed("config") {
		localConfig.Config.Value = cmd.Flag("config").Value.String()
		localConfig.Config.Scope = ""
	}

	return localConfig
}

// AllConfigs get all configs we know about
func AllConfigs() map[string]Config {
	return configContents
}

// Set a local config
func Set(scope string, options []string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.Err(err)
		}
	}

	for _, option := range options {
		optionArr := strings.Split(option, "=")
		key := optionArr[0]
		if len(optionArr) < 2 || (key != "key" && key != "project" && key != "config") {
			utils.Err(errors.New("invalid option " + option))
		}
	}

	for _, option := range options {
		optionArr := strings.Split(option, "=")
		key := optionArr[0]
		value := optionArr[1]

		if key == "key" {
			scopedConfig := configContents[scope]
			scopedConfig.Key = value
			configContents[scope] = scopedConfig
		}
		if key == "project" {
			scopedConfig := configContents[scope]
			scopedConfig.Project = value
			configContents[scope] = scopedConfig
		}
		if key == "config" {
			scopedConfig := configContents[scope]
			scopedConfig.Config = value
			configContents[scope] = scopedConfig
		}
	}

	writeYAML(configContents)
}

// Unset a local config
func Unset(scope string, options []string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.Err(err)
		}
	}

	for _, key := range options {
		if key != "key" && key != "project" && key != "config" {
			utils.Err(errors.New("invalid option " + key))
		}
	}

	if configContents[scope] == (Config{}) {
		return
	}

	for _, key := range options {
		if key == "key" {
			scopedConfig := configContents[scope]
			scopedConfig.Key = ""
			configContents[scope] = scopedConfig
		}
		if key == "project" {
			scopedConfig := configContents[scope]
			scopedConfig.Project = ""
			configContents[scope] = scopedConfig
		}
		if key == "config" {
			scopedConfig := configContents[scope]
			scopedConfig.Config = ""
			configContents[scope] = scopedConfig
		}
	}

	if configContents[scope] == (Config{}) {
		delete(configContents, scope)
	}

	writeYAML(configContents)
}

// Write config to ~/.doppler.yaml
func writeYAML(config map[string]Config) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		utils.Err(err)
	}

	err = ioutil.WriteFile(yamlFile, bytes, os.FileMode(0600))
	if err != nil {
		utils.Err(err)
	}
}

func exists() bool {
	return utils.Exists(yamlFile)
}

func readYAML() map[string]Config {
	fileContents, err := ioutil.ReadFile(yamlFile)
	if err != nil {
		utils.Err(err)
	}

	var config map[string]Config
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
