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
	"gopkg.in/gookit/color.v1"
)

type tableOptions struct {
	Title           string
	ShowBorder      bool
	SeparateHeader  bool
	SeparateColumns bool
}

const maxTableWidth = 100

// TableOptions customize table display
func TableOptions() tableOptions {
	return tableOptions{ShowBorder: true, SeparateHeader: true, SeparateColumns: true}
}

// Table print table
func Table(headers []string, rows [][]string, options tableOptions) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	t.SetTitle(options.Title)
	t.Style().Options.DrawBorder = options.ShowBorder
	t.Style().Options.SeparateHeader = options.SeparateHeader
	t.Style().Options.SeparateColumns = options.SeparateColumns

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

// ConfigLogs print config logs
func ConfigLogs(logs []models.ConfigLog, number int, jsonFlag bool) {
	maxLogs := int(math.Min(float64(len(logs)), float64(number)))
	logs = logs[0:maxLogs]

	if jsonFlag {
		JSON(logs)
		return
	}

	for _, log := range logs {
		ConfigLog(log, false, false)
	}
}

// ConfigLog print config log
func ConfigLog(log models.ConfigLog, jsonFlag bool, diff bool) {
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

	if diff && len(log.Diff) > 0 {
		fmt.Println("")

		for i, logDiff := range log.Diff {
			if i != 0 {
				fmt.Print("\n")
			}

			if logDiff.Name == "" {
				color.Red.Println(logDiff.Removed)
				color.Green.Println(logDiff.Added)
			} else {
				color.Red.Println("-", logDiff.Name, "=", logDiff.Removed)
				color.Green.Println("+", logDiff.Name, "=", logDiff.Added)
			}
		}
	}
}

// ActivityLogs print activity logs
func ActivityLogs(logs []models.ActivityLog, number int, jsonFlag bool) {
	maxLogs := int(math.Min(float64(len(logs)), float64(number)))
	logs = logs[0:maxLogs]

	if jsonFlag {
		JSON(logs)
		return
	}

	for _, log := range logs {
		ActivityLog(log, false, false)
	}
}

// ActivityLog print activity log
func ActivityLog(log models.ActivityLog, jsonFlag bool, diff bool) {
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

	rows := [][]string{{info.Name, info.InitialFetchAt, info.LastFetchAt, info.CreatedAt, info.Environment, info.Project}}
	Table([]string{"name", "initial fetch", "last fetch", "created at", "stage", "project"}, rows, TableOptions())
}

// ConfigsInfo print configs
func ConfigsInfo(info []models.ConfigInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, configInfo := range info {
		rows = append(rows, []string{configInfo.Name, configInfo.InitialFetchAt, configInfo.LastFetchAt, configInfo.CreatedAt,
			configInfo.Environment, configInfo.Project})
	}
	Table([]string{"name", "initial fetch", "last fetch", "created at", "stage", "project"}, rows, TableOptions())
}

// EnvironmentsInfo print environments
func EnvironmentsInfo(info []models.EnvironmentInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, environmentInfo := range info {
		rows = append(rows, []string{environmentInfo.ID, environmentInfo.Name, environmentInfo.InitialFetchAt,
			environmentInfo.CreatedAt, environmentInfo.Project})
	}
	Table([]string{"id", "name", "initial fetch", "created at", "project"}, rows, TableOptions())
}

// EnvironmentInfo print environment
func EnvironmentInfo(info models.EnvironmentInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.ID, info.Name, info.InitialFetchAt, info.CreatedAt, info.Project}}
	Table([]string{"id", "name", "initial fetch", "created at", "project"}, rows, TableOptions())
}

// ProjectsInfo print info of multiple projects
func ProjectsInfo(info []models.ProjectInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	var rows [][]string
	for _, projectInfo := range info {
		rows = append(rows, []string{projectInfo.ID, projectInfo.Name, projectInfo.Description, projectInfo.CreatedAt})
	}
	Table([]string{"id", "name", "description", "created at"}, rows, TableOptions())
}

// ProjectInfo print project info
func ProjectInfo(info models.ProjectInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.ID, info.Name, info.Description, info.CreatedAt}}
	Table([]string{"id", "name", "description", "created at"}, rows, TableOptions())
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

	Table(headers, rows, TableOptions())
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
	Table([]string{"name"}, rows, TableOptions())
}

// Settings print settings
func Settings(settings models.WorkplaceSettings, jsonFlag bool) {
	if jsonFlag {
		JSON(settings)
		return
	}

	rows := [][]string{{settings.ID, settings.Name, settings.BillingEmail}}
	Table([]string{"id", "name", "billing email"}, rows, TableOptions())
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

	options := TableOptions()
	options.Title = title
	Table(headers, rows, options)
}

// ScopedConfigValues print scoped config value(s)
func ScopedConfigValues(conf models.ScopedOptions, args []string, values map[string]*models.ScopedOption, jsonFlag bool, plain bool) {
	if plain {
		sbEmpty := true
		var sb strings.Builder

		for _, arg := range args {
			if sbEmpty {
				sbEmpty = false
			} else {
				sb.WriteString("\n")
			}

			if option, exists := values[arg]; exists {
				sb.WriteString(option.Value)
			}
		}

		fmt.Println(sb.String())
		return
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
			rows = append(rows, []string{arg, option.Value, option.Scope})
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

	Table([]string{"name", "value", "scope"}, rows, TableOptions())
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
	Table([]string{"name"}, rows, TableOptions())
}

// ConfigServiceTokensInfo print info of multiple config service tokens
func ConfigServiceTokensInfo(tokens []models.ConfigServiceToken, number int, jsonFlag bool) {
	maxTokens := int(math.Min(float64(len(tokens)), float64(number)))
	tokens = tokens[0:maxTokens]

	if jsonFlag {
		JSON(tokens)
		return
	}

	rows := [][]string{}
	for _, token := range tokens {
		rows = append(rows, []string{token.Name, token.Slug, token.CreatedAt})
	}
	Table([]string{"name", "slug", "created at"}, rows, TableOptions())
}

// ConfigServiceTokenInfo print config service token info
func ConfigServiceTokenInfo(token models.ConfigServiceToken, jsonFlag bool) {
	if jsonFlag {
		JSON(token)
		return
	}

	ConfigServiceTokensInfo([]models.ConfigServiceToken{token}, 1, false)
}

// ConfigServiceToken print config service token and its info
func ConfigServiceToken(token models.ConfigServiceToken, jsonFlag bool) {
	if jsonFlag {
		JSON(token)
		return
	}

	rows := [][]string{{token.Name, token.Token, token.Slug, token.CreatedAt}}
	Table([]string{"name", "token", "slug", "created at"}, rows, TableOptions())
}
