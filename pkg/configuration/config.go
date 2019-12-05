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
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// UserConfigPath path to the user's configuration file
var UserConfigPath string

var configContents models.ConfigFile

func init() {
	fileName := ".doppler.yaml"
	configDir := utils.ConfigDir()
	if utils.Exists(configDir) {
		UserConfigPath = filepath.Join(configDir, fileName)
	} else {
		UserConfigPath = filepath.Join(utils.HomeDir(), fileName)
	}

	if !exists() {
		if jsonExists() {
			migrateJSONToYaml()
		} else {
			var blankConfig models.ConfigFile
			writeYAML(blankConfig)
		}
	}
}

// LoadConfig load the configuration file
func LoadConfig() {
	configContents = readYAML()
}

// Get the config at the specified scope
func Get(scope string) models.ScopedOptions {
	scope, err := parseScope(scope)
	if err != nil {
		utils.Err(err)
	}
	scope = filepath.Clean(scope) + string(filepath.Separator)
	var scopedConfig models.ScopedOptions

	for confScope, conf := range configContents.ScopedOptions {
		// both paths should end in / to prevent partial match (e.g. /test matching /test123)
		if confScope != "*" && !strings.HasPrefix(scope, filepath.Clean(confScope)+string(filepath.Separator)) {
			continue
		}

		pairs := models.Pairs(conf)
		scopedPairs := models.ScopedPairs(&scopedConfig)
		for name, pair := range pairs {
			if pair != "" {
				scopedPair := scopedPairs[name]
				if *scopedPair == (models.ScopedOption{}) || len(confScope) > len(scopedPair.Scope) {
					scopedPair.Value = pair
					scopedPair.Scope = confScope
					scopedPair.Source = models.ConfigFileSource.String()
				}
			}
		}
	}

	return scopedConfig
}

// LocalConfig retrieves the config for the scoped directory
func LocalConfig(cmd *cobra.Command) models.ScopedOptions {
	// cli config file (lowest priority)
	localConfig := Get(cmd.Flag("scope").Value.String())

	// environment variables
	if !utils.GetBoolFlag(cmd, "no-read-env") {
		pairs := models.EnvPairs(&localConfig)
		for envVar, pair := range pairs {
			envValue := os.Getenv(envVar)
			if envValue != "" {
				pair.Value = envValue
				pair.Scope = "*"
				pair.Source = models.EnvironmentSource.String()
			}
		}
	}

	// individual flags (highest priority)
	flagSet := cmd.Flags().Changed("token")
	if flagSet || localConfig.Token.Value == "" {
		localConfig.Token.Value = cmd.Flag("token").Value.String()
		localConfig.Token.Scope = "*"

		if flagSet {
			localConfig.Token.Source = models.FlagSource.String()
		} else {
			localConfig.Token.Source = models.DefaultValueSource.String()
		}
	}

	flagSet = cmd.Flags().Changed("api-host")
	if flagSet || localConfig.APIHost.Value == "" {
		localConfig.APIHost.Value = cmd.Flag("api-host").Value.String()
		localConfig.APIHost.Scope = "*"

		if flagSet {
			localConfig.APIHost.Source = models.FlagSource.String()
		} else {
			localConfig.APIHost.Source = models.DefaultValueSource.String()
		}
	}

	flagSet = cmd.Flags().Changed("dashboard-host")
	if flagSet || localConfig.DashboardHost.Value == "" {
		localConfig.DashboardHost.Value = cmd.Flag("dashboard-host").Value.String()
		localConfig.DashboardHost.Scope = "*"

		if flagSet {
			localConfig.DashboardHost.Source = models.FlagSource.String()
		} else {
			localConfig.DashboardHost.Source = models.DefaultValueSource.String()
		}
	}

	flagSet = cmd.Flags().Changed("no-verify-tls")
	if flagSet || localConfig.VerifyTLS.Value == "" {
		noVerifyTLS := cmd.Flag("no-verify-tls").Value.String()
		localConfig.VerifyTLS.Value = strconv.FormatBool(!utils.GetBool(noVerifyTLS, false))
		localConfig.VerifyTLS.Scope = "*"

		if flagSet {
			localConfig.VerifyTLS.Source = models.FlagSource.String()
		} else {
			localConfig.VerifyTLS.Source = models.DefaultValueSource.String()
		}
	}

	// these flags below don't have a default value and should only be used if specified by the user (or will cause invalid memory access)
	flagSet = cmd.Flags().Changed("project")
	if flagSet {
		localConfig.Project.Value = cmd.Flag("project").Value.String()
		localConfig.Project.Scope = "*"

		if flagSet {
			localConfig.Project.Source = models.FlagSource.String()
		} else {
			localConfig.Project.Source = models.DefaultValueSource.String()
		}
	}

	flagSet = cmd.Flags().Changed("config")
	if flagSet {
		localConfig.Config.Value = cmd.Flag("config").Value.String()
		localConfig.Config.Scope = "*"

		if flagSet {
			localConfig.Config.Source = models.FlagSource.String()
		} else {
			localConfig.Config.Source = models.DefaultValueSource.String()
		}
	}

	return localConfig
}

// AllConfigs get all configs we know about
func AllConfigs() map[string]models.FileScopedOptions {
	return configContents.ScopedOptions
}

// Set properties on a scoped config
func Set(scope string, options map[string]string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.Err(err)
		}
	}

	for key, value := range options {
		if !IsValidConfigOption(key) {
			utils.Err(errors.New("invalid option "+key), "")
		}

		config := configContents.ScopedOptions[scope]
		SetConfigValue(&config, key, value)
		configContents.ScopedOptions[scope] = config
	}

	writeYAML(configContents)
}

// SetFromConfig set properties on a scoped config using a config object
func SetFromConfig(scope string, config models.FileScopedOptions) {
	pairs := models.Pairs(config)
	Set(scope, pairs)
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

	if configContents.ScopedOptions[scope] == (models.FileScopedOptions{}) {
		return
	}

	for _, key := range options {
		if !IsValidConfigOption(key) {
			utils.Err(errors.New("invalid option "+key), "")
		}

		config := configContents.ScopedOptions[scope]
		SetConfigValue(&config, key, "")
		configContents.ScopedOptions[scope] = config
	}

	if configContents.ScopedOptions[scope] == (models.FileScopedOptions{}) {
		delete(configContents.ScopedOptions, scope)
	}

	writeYAML(configContents)
}

// Write config to filesystem
func writeYAML(config models.ConfigFile) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		utils.Err(err)
	}

	err = ioutil.WriteFile(UserConfigPath, bytes, os.FileMode(0600))
	if err != nil {
		utils.Err(err)
	}
}

func exists() bool {
	return utils.Exists(UserConfigPath)
}

func readYAML() models.ConfigFile {
	fileContents, err := ioutil.ReadFile(UserConfigPath)
	if err != nil {
		utils.Err(err)
	}

	var config models.ConfigFile
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

// IsValidConfigOption whether the specified key is a valid config option
func IsValidConfigOption(key string) bool {
	configOptions := map[string]interface{}{
		"token":          nil,
		"project":        nil,
		"config":         nil,
		"api-host":       nil,
		"dashboard-host": nil,
		"verify-tls":     nil,
	}

	_, exists := configOptions[key]
	return exists
}

// GetScopedConfigValue get the value of the specified key within the config
func GetScopedConfigValue(conf models.ScopedOptions, key string) (string, string) {
	pairs := models.ScopedPairs(&conf)
	for name, pair := range pairs {
		if key == name {
			return pair.Value, pair.Scope
		}
	}

	return "", ""
}

// SetConfigValue set the value for the specified key in the config
func SetConfigValue(conf *models.FileScopedOptions, key string, value string) {
	if key == "token" {
		(*conf).Token = value
	} else if key == "project" {
		(*conf).Project = value
	} else if key == "config" {
		(*conf).Config = value
	} else if key == "api-host" {
		(*conf).APIHost = value
	} else if key == "dashboard-host" {
		(*conf).DashboardHost = value
	} else if key == "verify-tls" {
		(*conf).VerifyTLS = value
	}
}
