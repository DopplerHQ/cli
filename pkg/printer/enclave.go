/*
Copyright © 2021 Doppler <support@doppler.com>

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
	"sort"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"gopkg.in/gookit/color.v1"
)

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
				fmt.Println("")
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
		utils.HandleError(err)
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
	Table([]string{"name", "initial fetch", "last fetch", "created at", "environment", "project"}, rows, TableOptions())
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
	Table([]string{"name", "initial fetch", "last fetch", "created at", "environment", "project"}, rows, TableOptions())
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
func Secrets(secrets map[string]models.ComputedSecret, secretsToPrint []string, jsonFlag bool, plain bool, raw bool, copy bool, visibility bool) {
	if len(secretsToPrint) == 0 {
		for name := range secrets {
			secretsToPrint = append(secretsToPrint, name)
		}
		sort.Strings(secretsToPrint)
	}

	if copy {
		vals := []string{}
		for _, name := range secretsToPrint {
			if secrets[name] != (models.ComputedSecret{}) {
				if secrets[name].ComputedValue == nil {
					utils.HandleError(fmt.Errorf("Unable to copy restricted value to clipboard"))
				} else {
					vals = append(vals, *secrets[name].ComputedValue)
				}
			}
		}

		if err := utils.CopyToClipboard(strings.Join(vals, "\n")); err != nil {
			utils.HandleError(err, "Unable to copy to clipboard")
		}
	}

	if jsonFlag {
		secretsMap := map[string]map[string]interface{}{}
		for _, name := range secretsToPrint {
			if secrets[name] != (models.ComputedSecret{}) {
				secretsMap[name] = map[string]interface{}{
					"note":               secrets[name].Note,
					"computedVisibility": secrets[name].ComputedVisibility,
				}

				if secrets[name].ComputedValue != nil {
					secretsMap[name]["computed"] = *secrets[name].ComputedValue
				} else {
					secretsMap[name]["computed"] = nil
				}

				if raw {
					secretsMap[name]["rawVisibility"] = secrets[name].RawVisibility
					if secrets[name].RawValue != nil {
						secretsMap[name]["raw"] = *secrets[name].RawValue
					} else {
						secretsMap[name]["raw"] = nil
					}
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
		vals := []string{}
		for _, secret := range matchedSecrets {
			if raw {
				if secret.RawValue != nil {
					vals = append(vals, *secret.RawValue)
				} else {
					vals = append(vals, "")
				}
			} else {
				if secret.ComputedValue != nil {
					vals = append(vals, *secret.ComputedValue)
				} else {
					vals = append(vals, "")
				}
			}
		}

		fmt.Println(strings.Join(vals, "\n"))
		return
	}

	headers := []string{"name"}
	if visibility {
		headers = append(headers, "visibility")
	}
	headers = append(headers, "value")
	if raw {
		if visibility {
			headers = append(headers, "raw visibility")
		}
		headers = append(headers, "raw value")
	}
	headers = append(headers, "note")

	var rows [][]string
	for _, secret := range matchedSecrets {
		var computedValue string
		if secret.ComputedValue != nil {
			computedValue = *secret.ComputedValue
		} else {
			computedValue = "[RESTRICTED]"
		}

		row := []string{secret.Name}
		if visibility {
			row = append(row, secret.ComputedVisibility)
		}
		row = append(row, computedValue)
		if raw {
			var rawValue string
			if secret.RawValue != nil {
				rawValue = *secret.RawValue
			} else {
				rawValue = "[RESTRICTED]"
			}
			if visibility {
				row = append(row, secret.RawVisibility)
			}
			row = append(row, rawValue)
		}

		row = append(row, secret.Note)

		rows = append(rows, row)
	}

	Table(headers, rows, TableOptions())
}

// SecretsNames print secrets names
func SecretsNames(secretsNames []string, jsonFlag bool) {
	if jsonFlag {
		secretsMap := map[string]map[string]string{}
		for _, name := range secretsNames {
			secretsMap[name] = map[string]string{}
		}

		JSON(secretsMap)
		return
	}

	var rows [][]string
	for _, name := range secretsNames {
		rows = append(rows, []string{name})
	}
	Table([]string{"name"}, rows, TableOptions())
}

// SecretNote print a secret's note
func SecretNote(secret models.SecretNote, jsonFlag bool) {
	if jsonFlag {
		note := map[string]string{}
		note["name"] = secret.Secret
		note["note"] = secret.Note

		JSON(note)
		return
	}

	var rows [][]string
	rows = append(rows, []string{secret.Secret, secret.Note})
	Table([]string{"name", "note"}, rows, TableOptions())
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
		rows = append(rows, []string{token.Name, token.Slug, token.Project, token.Environment, token.Config, token.CreatedAt, token.ExpiresAt, token.Access})
	}
	Table([]string{"name", "slug", "project", "environment", "config", "created at", "expires at", "access"}, rows, TableOptions())
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
func ConfigServiceToken(token models.ConfigServiceToken, jsonFlag bool, plain bool, copy bool) {
	if copy {
		if err := utils.CopyToClipboard(token.Token); err != nil {
			utils.HandleError(err, "Unable to copy to clipboard")
		}
	}

	if plain {
		fmt.Println(token.Token)
		return
	}

	if jsonFlag {
		JSON(token)
		return
	}

	rows := [][]string{{token.Name, token.Token, token.Slug, token.Project, token.Environment, token.Config, token.CreatedAt, token.ExpiresAt, token.Access}}
	Table([]string{"name", "token", "slug", "project", "environment", "config", "created at", "expires at", "access"}, rows, TableOptions())
}

func ActorInfo(info models.ActorInfo, jsonFlag bool) {
	if jsonFlag {
		JSON(info)
		return
	}

	rows := [][]string{{info.Name, info.Type, fmt.Sprintf("%s (%s)", info.Workplace.Name, info.Workplace.Slug), info.TokenPreview, info.Slug, info.CreatedAt, info.LastSeenAt}}
	Table([]string{"name", "type", "workplace", "token preview", "slug", "created at", "last seen at"}, rows, TableOptions())
}
