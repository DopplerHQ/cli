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
	"doppler-cli/models"
	utils "doppler-cli/utils"
	"encoding/json"

	"github.com/spf13/cobra"
)

// GetAPISecrets for specified project and config
func GetAPISecrets(cmd *cobra.Command, apiKey string, project string, config string) ([]byte, map[string]models.ComputedSecret) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/variables", params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch secrets")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	computed := make(map[string]models.ComputedSecret)
	secrets := result["variables"].(map[string]interface{})
	// fmt.Println("secret1", secrets)
	for key, secret := range secrets {
		val := secret.(map[string]interface{})
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return response, computed
}

// SetAPISecrets for specified project and config
func SetAPISecrets(cmd *cobra.Command, apiKey string, project string, config string, secrets map[string]interface{}) ([]byte, map[string]models.ComputedSecret) {
	reqBody := make(map[string]interface{})
	reqBody["variables"] = secrets
	body, err := json.Marshal(reqBody)
	if err != nil {
		utils.Err(err, "Invalid secrets")
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/variables", params, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to fetch secrets")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	computed := make(map[string]models.ComputedSecret)
	for key, secret := range result["variables"].(map[string]interface{}) {
		val := secret.(map[string]interface{})
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return response, computed
}

// GetAPIWorkplaceSettings get specified workplace settings
func GetAPIWorkplaceSettings(cmd *cobra.Command, apiKey string) ([]byte, models.WorkplaceSettings) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/workplace", []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch workplace settings")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	settings := models.ParseWorkplaceSettings(result["workplace"].(map[string]interface{}))
	return response, settings
}

// SetAPIWorkplaceSettings set workplace settings
func SetAPIWorkplaceSettings(cmd *cobra.Command, apiKey string, values models.WorkplaceSettings) ([]byte, models.WorkplaceSettings) {
	body, err := json.Marshal(values)
	if err != nil {
		utils.Err(err, "Invalid workplace settings")
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/workplace", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to update workplace settings")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	settings := models.ParseWorkplaceSettings(result["workplace"].(map[string]interface{}))
	return response, settings
}

// GetAPIProjects get projects
func GetAPIProjects(cmd *cobra.Command, apiKey string) ([]byte, []models.ProjectInfo) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/pipelines", []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch projects")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	var info []models.ProjectInfo
	for _, project := range result["pipelines"].([]interface{}) {
		projectInfo := models.ParseProjectInfo(project.(map[string]interface{}))
		info = append(info, projectInfo)
	}
	return response, info
}

// GetAPIProject get specified project
func GetAPIProject(cmd *cobra.Command, apiKey string, project string) ([]byte, models.ProjectInfo) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch project")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// CreateAPIProject create a project
func CreateAPIProject(cmd *cobra.Command, apiKey string, name string, description string) ([]byte, models.ProjectInfo) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		utils.Err(err, "Invalid project info")
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/pipelines/", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to create project")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// UpdateAPIProject update a project
func UpdateAPIProject(cmd *cobra.Command, apiKey string, project string, name string, description string) ([]byte, models.ProjectInfo) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		utils.Err(err, "Invalid project info")
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to update project")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	projectInfo := models.ParseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// DeleteAPIProject create a project
func DeleteAPIProject(cmd *cobra.Command, apiKey string, project string) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.DeleteRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to delete project")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}
}

// GetAPIEnvironments get environments
func GetAPIEnvironments(cmd *cobra.Command, apiKey string, project string) ([]byte, []models.EnvironmentInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/stages", params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch environments")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	var info []models.EnvironmentInfo
	for _, environment := range result["stages"].([]interface{}) {
		environmentInfo := models.ParseEnvironmentInfo(environment.(map[string]interface{}))
		info = append(info, environmentInfo)
	}
	return response, info
}

// GetAPIEnvironment get specified environment
func GetAPIEnvironment(cmd *cobra.Command, apiKey string, project string, environment string) ([]byte, models.EnvironmentInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/stages/"+environment, params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch environment")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	info := models.ParseEnvironmentInfo(result["stage"].(map[string]interface{}))
	return response, info
}

// GetAPIConfigs get configs
func GetAPIConfigs(cmd *cobra.Command, apiKey string, project string) ([]byte, []models.ConfigInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/environments", params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch configs")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	var info []models.ConfigInfo
	for _, config := range result["environments"].([]interface{}) {
		configInfo := models.ParseConfigInfo(config.(map[string]interface{}))
		info = append(info, configInfo)
	}
	return response, info
}

// GetAPIConfig get a config
func GetAPIConfig(cmd *cobra.Command, apiKey string, project string, config string) ([]byte, models.ConfigInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/environments/"+config, params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch configs")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return response, info
}

// CreateAPIConfig create a config
func CreateAPIConfig(cmd *cobra.Command, apiKey string, project string, name string, environment string, defaults bool) ([]byte, models.ConfigInfo) {
	postBody := map[string]interface{}{"name": name, "stage": environment, "defaults": defaults}
	body, err := json.Marshal(postBody)
	if err != nil {
		utils.Err(err, "Invalid config info")
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/environments", params, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to create config")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return response, info
}

// DeleteAPIConfig create a config
func DeleteAPIConfig(cmd *cobra.Command, apiKey string, project string, config string) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.DeleteRequest(host, "v2/environments/"+config, params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to delete config")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}
}

// UpdateAPIConfig create a config
func UpdateAPIConfig(cmd *cobra.Command, apiKey string, project string, config string, name string) ([]byte, models.ConfigInfo) {
	postBody := map[string]interface{}{"name": name}
	body, err := json.Marshal(postBody)
	if err != nil {
		utils.Err(err, "Invalid config info")
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/environments/"+config, params, apiKey, body)
	if err != nil {
		utils.Err(err, "Unable to update config")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	info := models.ParseConfigInfo(result["environment"].(map[string]interface{}))
	return response, info
}

// GetAPIActivityLogs get activity logs
func GetAPIActivityLogs(cmd *cobra.Command, apiKey string) ([]byte, []models.Log) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/logs", []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch activity logs")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	var logs []models.Log
	for _, log := range result["logs"].([]interface{}) {
		parsedLog := models.ParseLog(log.(map[string]interface{}))
		logs = append(logs, parsedLog)
	}
	return response, logs
}

// GetAPIActivityLog get specified activity log
func GetAPIActivityLog(cmd *cobra.Command, apiKey string, log string) ([]byte, models.Log) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/logs/"+log, []utils.QueryParam{}, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch activity log")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return response, parsedLog
}

// GetAPIConfigLogs get config audit logs
func GetAPIConfigLogs(cmd *cobra.Command, apiKey string, project string, config string) ([]byte, []models.Log) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/environments/"+config+"/logs", params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch config logs")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	var logs []models.Log
	for _, log := range result["logs"].([]interface{}) {
		parsedLog := models.ParseLog(log.(map[string]interface{}))
		logs = append(logs, parsedLog)
	}
	return response, logs
}

// GetAPIConfigLog get config audit log
func GetAPIConfigLog(cmd *cobra.Command, apiKey string, project string, config string, log string) ([]byte, models.Log) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/environments/"+config+"/logs/"+log, params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to fetch config log")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return response, parsedLog
}

// RollbackAPIConfigLog rollback a config log
func RollbackAPIConfigLog(cmd *cobra.Command, apiKey string, project string, config string, log string) ([]byte, models.Log) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/environments/"+config+"/logs/"+log+"/rollback", params, apiKey, []byte{})
	if err != nil {
		utils.Err(err, "Unable to rollback config log")
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err, "")
	}

	parsedLog := models.ParseLog(result["log"].(map[string]interface{}))
	return response, parsedLog
}
