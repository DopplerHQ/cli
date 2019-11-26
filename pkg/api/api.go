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
package api

import (
	"encoding/json"
	"strconv"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
	"github.com/spf13/cobra"
)

// Error API errors
type Error struct {
	Err     error
	Message string
}

// Unwrap get the original error
func (e *Error) Unwrap() error { return e.Err }

// IsNil whether the error is nil
func (e *Error) IsNil() bool { return e.Err == nil && e.Message == "" }

// GenerateAuthCode generate an auth code
func GenerateAuthCode(cmd *cobra.Command, host string, hostname string, os string, arch string) (map[string]interface{}, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "hostname", Value: hostname})
	params = append(params, utils.QueryParam{Key: "version", Value: version.ProgramVersion})
	params = append(params, utils.QueryParam{Key: "os", Value: os})
	params = append(params, utils.QueryParam{Key: "arch", Value: arch})

	response, err := utils.GetRequest(host, nil, "/auth/v1/cli/generate", params, "")
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth code"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	return result, Error{}
}

// GetAuthToken get an auth token
func GetAuthToken(cmd *cobra.Command, host string, code string) (map[string]interface{}, Error) {
	reqBody := make(map[string]interface{})
	reqBody["code"] = code
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth code"}
	}

	response, err := utils.PostRequest(host, nil, "/auth/v1/cli/authorize", []utils.QueryParam{}, "", body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth code"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth token"}
	}

	return result, Error{}
}

// RollAuthToken roll an auth token
func RollAuthToken(cmd *cobra.Command, host string, token string) (map[string]interface{}, Error) {
	reqBody := make(map[string]interface{})
	reqBody["token"] = token
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth token"}
	}

	response, err := utils.PostRequest(host, nil, "/auth/v1/cli/roll", []utils.QueryParam{}, "", body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to roll auth token"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	return result, Error{}
}

// RevokeAuthToken revoke an auth token
func RevokeAuthToken(cmd *cobra.Command, host string, token string) (map[string]interface{}, Error) {
	reqBody := make(map[string]interface{})
	reqBody["token"] = token
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth token"}
	}

	response, err := utils.PostRequest(host, nil, "/auth/v1/cli/revoke", []utils.QueryParam{}, "", body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to revoke auth token"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	return result, Error{}
}

// DownloadSecrets for specified project and config
func DownloadSecrets(cmd *cobra.Command, host string, apiKey string, project string, config string, metadata bool) ([]byte, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})
	params = append(params, utils.QueryParam{Key: "metadata", Value: strconv.FormatBool(metadata)})

	response, err := utils.GetRequest(host, map[string]string{"Accept": "text/plain"}, "/v2/variables", params, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to download secrets"}
	}

	return response, Error{}
}

// GetSecrets for specified project and config
func GetSecrets(cmd *cobra.Command, host string, apiKey string, project string, config string) ([]byte, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, map[string]string{"Accept": "application/json"}, "/v2/variables", params, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch secrets"}
	}

	return response, Error{}
}

// ParseSecrets for specified project and config
func ParseSecrets(response []byte) (map[string]models.ComputedSecret, Error) {
	var result map[string]interface{}
	err := json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	computed := make(map[string]models.ComputedSecret)
	secrets := result["variables"].(map[string]interface{})
	// fmt.Println("secret1", secrets)
	for key, secret := range secrets {
		val := secret.(map[string]interface{})
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return computed, Error{}
}

// SetSecrets for specified project and config
func SetSecrets(cmd *cobra.Command, host string, apiKey string, project string, config string, secrets map[string]interface{}) (map[string]models.ComputedSecret, Error) {
	reqBody := make(map[string]interface{})
	reqBody["variables"] = secrets
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid secrets"}
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.PostRequest(host, nil, "/v2/variables", params, apiKey, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to set secrets"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	computed := make(map[string]models.ComputedSecret)
	for key, secret := range result["variables"].(map[string]interface{}) {
		val := secret.(map[string]interface{})
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return computed, Error{}
}

// GetWorkplaceSettings get specified workplace settings
func GetWorkplaceSettings(cmd *cobra.Command, host string, apiKey string) (models.WorkplaceSettings, Error) {
	response, err := utils.GetRequest(host, nil, "/v2/workplace", []utils.QueryParam{}, apiKey)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to fetch workplace settings"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	settings := models.ParseWorkplaceSettings(result["workplace"].(map[string]interface{}))
	return settings, Error{}
}

// SetWorkplaceSettings set workplace settings
func SetWorkplaceSettings(cmd *cobra.Command, host string, apiKey string, values models.WorkplaceSettings) (models.WorkplaceSettings, Error) {
	body, err := json.Marshal(values)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Invalid workplace settings"}
	}

	response, err := utils.PostRequest(host, nil, "/v2/workplace", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to update workplace settings"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	settings := models.ParseWorkplaceSettings(result["workplace"].(map[string]interface{}))
	return settings, Error{}
}

// GetProjects get projects
func GetProjects(cmd *cobra.Command, host string, apiKey string) ([]models.ProjectInfo, Error) {
	response, err := utils.GetRequest(host, nil, "/v2/pipelines", []utils.QueryParam{}, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch projects"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	var info []models.ProjectInfo
	for _, project := range result["pipelines"].([]interface{}) {
		projectInfo := models.ParseProjectInfo(project.(map[string]interface{}))
		info = append(info, projectInfo)
	}
	return info, Error{}
}

// GetProject get specified project
func GetProject(cmd *cobra.Command, host string, apiKey string, project string) (models.ProjectInfo, Error) {
	response, err := utils.GetRequest(host, nil, "/v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to fetch project"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return projectInfo, Error{}
}

// CreateProject create a project
func CreateProject(cmd *cobra.Command, host string, apiKey string, name string, description string) (models.ProjectInfo, Error) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Invalid project info"}
	}

	response, err := utils.PostRequest(host, nil, "/v2/pipelines/", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to create project"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return projectInfo, Error{}
}

// UpdateProject update a project
func UpdateProject(cmd *cobra.Command, host string, apiKey string, project string, name string, description string) (models.ProjectInfo, Error) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Invalid project info"}
	}

	response, err := utils.PostRequest(host, nil, "/v2/pipelines/"+project, []utils.QueryParam{}, apiKey, body)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to update project"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return projectInfo, Error{}
}

// DeleteProject create a project
func DeleteProject(cmd *cobra.Command, host string, apiKey string, project string) Error {
	response, err := utils.DeleteRequest(host, nil, "/v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete project"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response"}
	}

	return Error{}
}

// GetEnvironments get environments
func GetEnvironments(cmd *cobra.Command, host string, apiKey string, project string) ([]models.EnvironmentInfo, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/stages", params, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch environments"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	var info []models.EnvironmentInfo
	for _, environment := range result["stages"].([]interface{}) {
		environmentInfo := models.ParseEnvironmentInfo(environment.(map[string]interface{}))
		info = append(info, environmentInfo)
	}
	return info, Error{}
}

// GetEnvironment get specified environment
func GetEnvironment(cmd *cobra.Command, host string, apiKey string, project string, environment string) (models.EnvironmentInfo, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/stages/"+environment, params, apiKey)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to fetch environment"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	info := models.ParseEnvironmentInfo(result["stage"].(map[string]interface{}))
	return info, Error{}
}

// GetConfigs get configs
func GetConfigs(cmd *cobra.Command, host string, apiKey string, project string) ([]models.ConfigInfo, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/environments", params, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch configs"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	var info []models.ConfigInfo
	for _, config := range result["environments"].([]interface{}) {
		configInfo := models.ParseConfigInfo(config.(map[string]interface{}))
		info = append(info, configInfo)
	}
	return info, Error{}
}

// GetConfig get a config
func GetConfig(cmd *cobra.Command, host string, apiKey string, project string, config string) (models.ConfigInfo, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/environments/"+config, params, apiKey)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to fetch configs"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return info, Error{}
}

// CreateConfig create a config
func CreateConfig(cmd *cobra.Command, host string, apiKey string, project string, name string, environment string, defaults bool) (models.ConfigInfo, Error) {
	postBody := map[string]interface{}{"name": name, "stage": environment, "defaults": defaults}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Invalid config info"}
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.PostRequest(host, nil, "/v2/environments", params, apiKey, body)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to create config"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return info, Error{}
}

// DeleteConfig create a config
func DeleteConfig(cmd *cobra.Command, host string, apiKey string, project string, config string) Error {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.DeleteRequest(host, nil, "/v2/environments/"+config, params, apiKey)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete config"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response"}
	}

	return Error{}
}

// UpdateConfig create a config
func UpdateConfig(cmd *cobra.Command, host string, apiKey string, project string, config string, name string) (models.ConfigInfo, Error) {
	postBody := map[string]interface{}{"name": name}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Invalid config info"}
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.PostRequest(host, nil, "/v2/environments/"+config, params, apiKey, body)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to update config"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return info, Error{}
}

// GetActivityLogs get activity logs
func GetActivityLogs(cmd *cobra.Command, host string, apiKey string) ([]models.Log, Error) {
	response, err := utils.GetRequest(host, nil, "/v2/logs", []utils.QueryParam{}, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch activity logs"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	var logs []models.Log
	for _, log := range result["logs"].([]interface{}) {
		parsedLog := models.ParseLog(log.(map[string]interface{}))
		logs = append(logs, parsedLog)
	}
	return logs, Error{}
}

// GetActivityLog get specified activity log
func GetActivityLog(cmd *cobra.Command, host string, apiKey string, log string) (models.Log, Error) {
	response, err := utils.GetRequest(host, nil, "/v2/logs/"+log, []utils.QueryParam{}, apiKey)
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to fetch activity log"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return parsedLog, Error{}
}

// GetConfigLogs get config audit logs
func GetConfigLogs(cmd *cobra.Command, host string, apiKey string, project string, config string) ([]models.Log, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/environments/"+config+"/logs", params, apiKey)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch config logs"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response"}
	}

	var logs []models.Log
	for _, log := range result["logs"].([]interface{}) {
		parsedLog := models.ParseLog(log.(map[string]interface{}))
		logs = append(logs, parsedLog)
	}
	return logs, Error{}
}

// GetConfigLog get config audit log
func GetConfigLog(cmd *cobra.Command, host string, apiKey string, project string, config string, log string) (models.Log, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, nil, "/v2/environments/"+config+"/logs/"+log, params, apiKey)
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to fetch config log"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return parsedLog, Error{}
}

// RollbackConfigLog rollback a config log
func RollbackConfigLog(cmd *cobra.Command, host string, apiKey string, project string, config string, log string) (models.Log, Error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.PostRequest(host, nil, "/v2/environments/"+config+"/logs/"+log+"/rollback", params, apiKey, []byte{})
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to rollback config log"}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.Log{}, Error{Err: err, Message: "Unable to parse API response"}
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return parsedLog, Error{}
}
