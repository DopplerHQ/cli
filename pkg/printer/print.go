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
package printer

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
)

const maxTableWidth = 100

// Table print table
func Table(headers []string, rows [][]string) {
	TableWithTitle(headers, rows, "")
}

// TableWithTitle print table with a title
func TableWithTitle(headers []string, rows [][]string, title string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	if title != "" {
		t.SetTitle(title)
	}

	tableHeaders := table.Row{}
	for _, header := range headers {
		tableHeaders = append(tableHeaders, header)
	}
	t.AppendHeader(tableHeaders)

	for _, row := range rows {
		tableRow := table.Row{}
		for _, val := range row {
			tableRow = append(tableRow, text.WrapText(val, maxTableWidth))
		}
		t.AppendRow(tableRow)
	}

	t.Render()
}

// Logs print logs
func Logs(logs []models.Log, number int, jsonFlag bool) {
	maxLogs := int(math.Min(float64(len(logs)), float64(number)))

	if jsonFlag {
		JSON(logs[0:maxLogs])
		return
	}

	for _, log := range logs[0:maxLogs] {
		Log(log, false)
	}
}

// Log print log
func Log(log models.Log, jsonFlag bool) {
	if jsonFlag {
		JSON(log)
		return
	}

	dateTime, err := time.Parse(time.RFC3339, log.CreatedAt)

	fmt.Println("Log " + log.ID)
	fmt.Println("User: " + log.User.Name + " <" + log.User.Email + ">")
	if err == nil {
		fmt.Println("Date: " + dateTime.In(time.Local).String())
	}
	fmt.Println("")
	fmt.Println("\t" + log.Text)
	fmt.Println("")
}

// JSON print object as json
func JSON(structure interface{}) {
	resp, err := json.Marshal(structure)
	if err != nil {
		utils.HandleError(err, "")
	}

	fmt.Println(string(resp))
}

// ConfigInfo print config
func ConfigInfo(info models.ConfigInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.Name, strings.Join(info.MissingVariables, ", "), info.DeployedAt, info.CreatedAt, info.Environment, info.Project}}
	Table([]string{"name", "missing_variables", "deployed_at", "created_at", "stage", "project"}, rows)
}

// ConfigsInfo print configs
func ConfigsInfo(info []models.ConfigInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, configInfo := range info {
		rows = append(rows, []string{configInfo.Name, strings.Join(configInfo.MissingVariables, ", "), configInfo.DeployedAt, configInfo.CreatedAt,
			configInfo.Environment, configInfo.Project})
	}
	Table([]string{"name", "missing_variables", "deployed_at", "created_at", "stage", "project"}, rows)
}

// EnvironmentsInfo print environments
func EnvironmentsInfo(info []models.EnvironmentInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, environmentInfo := range info {
		rows = append(rows, []string{environmentInfo.ID, environmentInfo.Name, environmentInfo.SetupAt, environmentInfo.FirstDeployAt,
			environmentInfo.CreatedAt, strings.Join(environmentInfo.MissingVariables, ", "), environmentInfo.Project})
	}
	Table([]string{"id", "name", "setup_at", "first_deploy_at", "created_at", "missing_variables", "project"}, rows)
}

// EnvironmentInfo print environment
func EnvironmentInfo(info models.EnvironmentInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.ID, info.Name, info.SetupAt, info.FirstDeployAt, info.CreatedAt, strings.Join(info.MissingVariables, ", "), info.Project}}
	Table([]string{"id", "name", "setup_at", "first_deploy_at", "created_at", "missing_variables", "project"}, rows)
}

// highest print projects info
// highest availability secrets storage on the planet
func ProjectsInfo(info []models.ProjectInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, projectInfo := range info {
		rows = append(rows, []string{projectInfo.ID, projectInfo.Name, projectInfo.Description, projectInfo.SetupAt, projectInfo.CreatedAt})
	}
	Table([]string{"id", "name", "description", "setup_at", "created_at"}, rows)
}

// ProjectInfo print project info
func ProjectInfo(info models.ProjectInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.ID, info.Name, info.Description, info.SetupAt, info.CreatedAt}}
	Table([]string{"id", "name", "description", "setup_at", "created_at"}, rows)
}

// Secrets print secrets
func Secrets(secrets map[string]models.ComputedSecret, secretsToPrint []string, jsonFlag bool, plain bool, raw bool) {
	if len(secretsToPrint) == 0 {
		for name := range secrets {
			secretsToPrint = append(secretsToPrint, name)
		}
		sort.Strings(secretsToPrint)
	}

	if jsonFlag {
		secretsMap := map[string]map[string]string{}
		for _, name := range secretsToPrint {
			if secrets[name] != (models.ComputedSecret{}) {
				secretsMap[name] = map[string]string{"computed": secrets[name].ComputedValue}
				if raw {
					secretsMap[name]["raw"] = secrets[name].RawValue
				}
			}
		}

		JSON(secretsMap)
		return
	}

	var matchedSecrets []models.ComputedSecret
	for _, name := range secretsToPrint {
		if secrets[name] != (models.ComputedSecret{}) {
			matchedSecrets = append(matchedSecrets, secrets[name])
		}
	}

	if plain {
		sbEmpty := true
		var sb strings.Builder
		for _, secret := range matchedSecrets {
			if sbEmpty {
				sbEmpty = false
			} else {
				sb.WriteString("\n")
			}

			if raw {
				sb.WriteString(secret.RawValue)
			} else {
				sb.WriteString(secret.ComputedValue)
			}
		}

		fmt.Println(sb.String())
		return
	}

	headers := []string{"name", "value"}
	if raw {
		headers = append(headers, "raw")
	}

	var rows [][]string
	for _, secret := range matchedSecrets {
		row := []string{secret.Name, secret.ComputedValue}
		if raw {
			row = append(row, secret.RawValue)
		}

		rows = append(rows, row)
	}

	Table(headers, rows)
}

// SecretsNames print secrets
func SecretsNames(secrets map[string]models.ComputedSecret, jsonFlag bool, plain bool) {
	var secretsNames []string
	for name := range secrets {
		secretsNames = append(secretsNames, name)
	}
	sort.Strings(secretsNames)

	if jsonFlag {
		secretsMap := map[string]map[string]string{}
		for _, name := range secretsNames {
			secretsMap[name] = map[string]string{}
		}

		JSON(secretsMap)
		return
	}

	if plain {
		sbEmpty := true
		var sb strings.Builder
		for _, name := range secretsNames {
			if sbEmpty {
				sbEmpty = false
			} else {
				sb.WriteString("\n")
			}

			sb.WriteString(name)
		}

		fmt.Println(sb.String())
		return
	}

	var rows [][]string
	for _, name := range secretsNames {
		rows = append(rows, []string{name})
	}
	Table([]string{"name"}, rows)
}

// Settings print settings
func Settings(settings models.WorkplaceSettings, jsonFlag bool) {
	if jsonFlag {
		JSON(settings)
		return
	}

	rows := [][]string{{settings.ID, settings.Name, settings.BillingEmail}}
	Table([]string{"id", "name", "billing_email"}, rows)
}

// ScopedConfig print scoped config
func ScopedConfig(conf models.ScopedOptions, jsonFlag bool) {
	ScopedConfigSource(conf, "", jsonFlag, false)
}

// ScopedConfigSource print scoped config with source
func ScopedConfigSource(conf models.ScopedOptions, title string, jsonFlag bool, source bool) {
	pairs := models.ScopedPairs(&conf)

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
			row := []string{name, pair.Value, pair.Scope}
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

	TableWithTitle(headers, rows, title)
}

// Configs print configs
func Configs(configs map[string]models.FileScopedOptions, jsonFlag bool) {
	if jsonFlag {
		JSON(configs)
		return
	}

	var rows [][]string
	for scope, conf := range configs {
		pairs := models.Pairs(conf)

		for name, value := range pairs {
			if value != "" {
				rows = append(rows, []string{name, value, scope})
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

	Table([]string{"name", "value", "scope"}, rows)
}

// ConfigOptionNames prints all supported config options
func ConfigOptionNames(options []string, jsonFlag bool) {
	if jsonFlag {
		JSON(options)
		return
	}

	rows := [][]string{}
	for _, option := range options {
		rows = append(rows, []string{option})
	}
	Table([]string{"name"}, rows)
}
