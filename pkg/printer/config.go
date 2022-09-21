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
package printer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

// ScopedConfig print scoped config
func ScopedConfig(conf models.ScopedOptions, jsonFlag bool) {
	ScopedConfigSource(conf, jsonFlag, false, true)
}

// ScopedConfigSource print scoped config with source
func ScopedConfigSource(conf models.ScopedOptions, jsonFlag bool, source bool, obfuscateToken bool) {
	pairs := models.ScopedOptionsMap(&conf)

	if jsonFlag {
		confMap := map[string]map[string]string{}

		for name, pair := range pairs {
			if *pair != (models.ScopedOption{}) {
				scope := pair.Scope
				value := pair.Value

				if confMap[scope] == nil {
					confMap[scope] = map[string]string{}
				}

				confMap[scope][name] = value
			}
		}

		JSON(confMap)
		return
	}

	var rows [][]string

	for name, pair := range pairs {
		if *pair != (models.ScopedOption{}) {
			translatedName := configuration.TranslateConfigOption(name)

			value := pair.Value
			if obfuscateToken && name == models.ConfigToken.String() {
				value = utils.RedactAuthToken(value)
			}

			row := []string{translatedName, value, pair.Scope}
			if source {
				row = append(row, pair.Source)
			}
			rows = append(rows, row)
		}
	}

	// sort by name
	sort.Slice(rows, func(a, b int) bool {
		return rows[a][0] < rows[b][0]
	})

	headers := []string{"name", "value", "scope"}
	if source {
		headers = append(headers, "source")
	}

	Table(headers, rows, TableOptions())
}

// ScopedConfigValues print scoped config value(s)
func ScopedConfigValues(conf models.ScopedOptions, args []string, values map[string]*models.ScopedOption, jsonFlag bool, plain bool, copy bool) {
	if plain || copy {
		vals := []string{}
		for _, arg := range args {
			if option, exists := values[arg]; exists {
				vals = append(vals, option.Value)
			}
		}

		print := strings.Join(vals, "\n")
		if copy {
			if err := utils.CopyToClipboard(print); err != nil {
				utils.HandleError(err, "Unable to copy to clipboard")
			}
		}

		if plain {
			fmt.Println(print)
			return
		}
	}

	if jsonFlag {
		filteredMap := map[string]string{}
		for _, arg := range args {
			if option, exists := values[arg]; exists {
				filteredMap[arg] = option.Value
			}
		}

		JSON(filteredMap)
		return
	}

	var rows [][]string
	for _, arg := range args {
		if option, exists := values[arg]; exists {
			translatedArg := configuration.TranslateConfigOption(arg)
			rows = append(rows, []string{translatedArg, option.Value, option.Scope})
		}
	}
	Table([]string{"name", "value", "scope"}, rows, TableOptions())
}

// Configs print configs
func Configs(configs map[string]models.FileScopedOptions, jsonFlag bool) {
	if jsonFlag {
		JSON(configs)
		return
	}

	var rows [][]string
	for scope, conf := range configs {
		pairs := models.OptionsMap(conf)

		for name, value := range pairs {
			if value != "" {
				translatedName := configuration.TranslateConfigOption(name)
				rows = append(rows, []string{translatedName, value, scope})
			}
		}
	}

	// sort by scope, then by name
	sort.Slice(rows, func(a, b int) bool {
		if rows[a][2] != rows[b][2] {
			return rows[a][2] < rows[b][2]
		}
		return rows[a][0] < rows[b][0]
	})

	Table([]string{"name", "value", "scope"}, rows, TableOptions())
}

// ConfigOptionNames prints all supported config options
func ConfigOptionNames(options []string, jsonFlag bool) {
	if jsonFlag {
		JSON(options)
		return
	}

	translatedOptions := []string{}
	for _, option := range options {
		translatedOptions = append(translatedOptions, configuration.TranslateConfigOption(option))
	}

	sort.Strings(translatedOptions)

	rows := [][]string{}
	for _, option := range translatedOptions {
		rows = append(rows, []string{option})
	}
	Table([]string{"name"}, rows, TableOptions())
}
