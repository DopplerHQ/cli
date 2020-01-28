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
	"fmt"
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
	configDir := utils.ConfigDir()
	if !utils.Exists(configDir) {
		configDir = utils.HomeDir()
	}

	fileName := ".doppler.yaml"
	filePath := filepath.Join(configDir, fileName)

	// support the cli v2 config path to allow downgrades
	cliV2Path := filepath.Join(utils.HomeDir(), ".doppler", fileName)
	if !utils.Exists(filePath) && utils.Exists(cliV2Path) {
		filePath = cliV2Path
	}

	UserConfigPath = filePath

	if !exists() {
		if jsonExists() {
			migrateJSONToYaml()
		} else {
			var blankConfig models.ConfigFile
			writeConfig(blankConfig)
		}
	}
}

// LoadConfig load the configuration file
func LoadConfig() {
	configContents = readConfig()
}

// VersionCheck the last version check
func VersionCheck() models.VersionCheck {
	return configContents.VersionCheck
}

// SetVersionCheck the last version check
func SetVersionCheck(version models.VersionCheck) {
	configContents.VersionCheck = version
	writeConfig(configContents)
}

// Get the config at the specified scope
func Get(scope string) models.ScopedOptions {
	scope, err := parseScope(scope)
	if err != nil {
		utils.HandleError(err)
	}
	scope = filepath.Clean(scope) + string(filepath.Separator)
	var scopedConfig models.ScopedOptions

	for confScope, conf := range configContents.Scoped {
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

	// these flags below do not have a default value and should only be used if specified by the user (or will cause invalid memory access)
	flagSet = cmd.Flags().Changed("project")
	if flagSet {
		localConfig.EnclaveProject.Value = cmd.Flag("project").Value.String()
		localConfig.EnclaveProject.Scope = "*"

		if flagSet {
			localConfig.EnclaveProject.Source = models.FlagSource.String()
		} else {
			localConfig.EnclaveProject.Source = models.DefaultValueSource.String()
		}
	}

	flagSet = cmd.Flags().Changed("config")
	if flagSet {
		localConfig.EnclaveConfig.Value = cmd.Flag("config").Value.String()
		localConfig.EnclaveConfig.Scope = "*"

		if flagSet {
			localConfig.EnclaveConfig.Source = models.FlagSource.String()
		} else {
			localConfig.EnclaveConfig.Source = models.DefaultValueSource.String()
		}
	}

	return localConfig
}

// AllConfigs get all configs we know about
func AllConfigs() map[string]models.FileScopedOptions {
	return configContents.Scoped
}

// Set properties on a scoped config
func Set(scope string, options map[string]string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.HandleError(err)
		}
	}

	for key, value := range options {
		if !IsValidConfigOption(key) {
			utils.HandleError(errors.New("invalid option "+key), "")
		}

		config := configContents.Scoped[scope]
		SetConfigValue(&config, key, value)
		configContents.Scoped[scope] = config
	}

	writeConfig(configContents)
}

// Unset a local config
func Unset(scope string, options []string) {
	if scope != "*" {
		var err error
		scope, err = parseScope(scope)
		if err != nil {
			utils.HandleError(err)
		}
	}

	if configContents.Scoped[scope] == (models.FileScopedOptions{}) {
		return
	}

	for _, key := range options {
		if !IsValidConfigOption(key) {
			utils.HandleError(errors.New("invalid option "+key), "")
		}

		config := configContents.Scoped[scope]
		SetConfigValue(&config, key, "")
		configContents.Scoped[scope] = config
	}

	if configContents.Scoped[scope] == (models.FileScopedOptions{}) {
		delete(configContents.Scoped, scope)
	}

	writeConfig(configContents)
}

// Write config to filesystem
func writeConfig(config models.ConfigFile) {
	bytes, err := yaml.Marshal(config)
	if err != nil {
		utils.HandleError(err)
	}

	utils.LogDebug(fmt.Sprintf("Writing user config to %s", UserConfigPath))
	err = ioutil.WriteFile(UserConfigPath, bytes, os.FileMode(0600))
	if err != nil {
		utils.HandleError(err)
	}
}

func exists() bool {
	return utils.Exists(UserConfigPath)
}

func readConfig() models.ConfigFile {
	utils.LogDebug(fmt.Sprintf("Reading user config from %s", UserConfigPath))

	fileContents, err := ioutil.ReadFile(UserConfigPath)
	if err != nil {
		utils.HandleError(err)
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
		models.ConfigToken.String():          nil,
		models.ConfigAPIHost.String():        nil,
		models.ConfigDashboardHost.String():  nil,
		models.ConfigVerifyTLS.String():      nil,
		models.ConfigEnclaveProject.String(): nil,
		models.ConfigEnclaveConfig.String():  nil,
	}

	_, exists := configOptions[key]
	return exists
}

// SetConfigValue set the value for the specified key in the config
func SetConfigValue(conf *models.FileScopedOptions, key string, value string) {
	if key == models.ConfigToken.String() {
		(*conf).Token = value
	} else if key == models.ConfigAPIHost.String() {
		(*conf).APIHost = value
	} else if key == models.ConfigDashboardHost.String() {
		(*conf).DashboardHost = value
	} else if key == models.ConfigVerifyTLS.String() {
		(*conf).VerifyTLS = value
	} else if key == models.ConfigEnclaveProject.String() {
		(*conf).EnclaveProject = value
	} else if key == models.ConfigEnclaveConfig.String() {
		(*conf).EnclaveConfig = value
	}
}
