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
package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/version"
)

// Error API errors
type Error struct {
	Err     error
	Message string
	Code    int
}

// Unwrap get the original error
func (e *Error) Unwrap() error { return e.Err }

// IsNil whether the error is nil
func (e *Error) IsNil() bool { return e.Err == nil && e.Message == "" }

func apiKeyHeader(apiKey string) map[string]string {
	encoded := base64.StdEncoding.EncodeToString([]byte(apiKey + ":"))
	return map[string]string{"Authorization": fmt.Sprintf("Basic %s", encoded)}
}

// GenerateAuthCode generate an auth code
func GenerateAuthCode(host string, verifyTLS bool, hostname string, os string, arch string) (map[string]interface{}, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "hostname", Value: hostname})
	params = append(params, queryParam{Key: "version", Value: version.ProgramVersion})
	params = append(params, queryParam{Key: "os", Value: os})
	params = append(params, queryParam{Key: "arch", Value: arch})

	statusCode, _, response, err := GetRequest(host, verifyTLS, nil, "/v3/auth/cli/generate/2", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth code", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return result, Error{}
}

// GetAuthToken get an auth token
func GetAuthToken(host string, verifyTLS bool, code string) (map[string]interface{}, Error) {
	reqBody := map[string]interface{}{}
	reqBody["code"] = code
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth code"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, nil, "/v3/auth/cli/authorize", []queryParam{}, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth token", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch auth token", Code: statusCode}
	}

	return result, Error{}
}

// RollAuthToken roll an auth token
func RollAuthToken(host string, verifyTLS bool, token string) (map[string]interface{}, Error) {
	reqBody := map[string]interface{}{}
	reqBody["token"] = token
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth token"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, nil, "/v3/auth/cli/roll", []queryParam{}, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to roll auth token", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return result, Error{}
}

// RevokeAuthToken revoke an auth token
func RevokeAuthToken(host string, verifyTLS bool, token string) (map[string]interface{}, Error) {
	reqBody := map[string]interface{}{}
	reqBody["token"] = token
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid auth token"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, nil, "/v3/auth/cli/revoke", []queryParam{}, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to revoke auth token", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return result, Error{}
}

// DownloadSecrets for specified project and config
func DownloadSecrets(host string, verifyTLS bool, apiKey string, project string, config string, format models.SecretsFormat, nameTransformer *models.SecretsNameTransformer, etag string, dynamicSecretsTTL time.Duration) (int, http.Header, []byte, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "format", Value: format.String()})
	params = append(params, queryParam{Key: "include_dynamic_secrets", Value: "true"})

	if dynamicSecretsTTL > 0 {
		ttlSeconds := int(dynamicSecretsTTL.Seconds())
		params = append(params, queryParam{Key: "dynamic_secrets_ttl_sec", Value: strconv.Itoa(ttlSeconds)})
	}
	if nameTransformer != nil {
		params = append(params, queryParam{Key: "name_transformer", Value: nameTransformer.Type})
	}

	headers := apiKeyHeader(apiKey)
	if etag != "" {
		headers["If-None-Match"] = etag
	}

	statusCode, respHeaders, response, err := GetRequest(host, verifyTLS, headers, "/v3/configs/config/secrets/download", params)
	if err != nil {
		return statusCode, respHeaders, nil, Error{Err: err, Message: "Unable to download secrets", Code: statusCode}
	}

	return statusCode, respHeaders, response, Error{}
}

// GetSecrets for specified project and config
func GetSecrets(host string, verifyTLS bool, apiKey string, project string, config string, secrets []string, includeDynamicSecrets bool, dynamicSecretsTTL time.Duration) ([]byte, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "include_dynamic_secrets", Value: strconv.FormatBool(includeDynamicSecrets)})

	if secrets != nil {
		params = append(params, queryParam{Key: "secrets", Value: strings.Join(secrets, ",")})
	}

	if dynamicSecretsTTL > 0 {
		ttlSeconds := int(dynamicSecretsTTL.Seconds())
		params = append(params, queryParam{Key: "dynamic_secrets_ttl_sec", Value: strconv.Itoa(ttlSeconds)})
	}

	headers := apiKeyHeader(apiKey)
	headers["Accept"] = "application/json"
	statusCode, _, response, err := GetRequest(host, verifyTLS, headers, "/v3/configs/config/secrets", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch secrets", Code: statusCode}
	}

	return response, Error{}
}

// SetSecrets for specified project and config
func SetSecrets(host string, verifyTLS bool, apiKey string, project string, config string, secrets map[string]interface{}) (map[string]models.ComputedSecret, Error) {
	reqBody := map[string]interface{}{}
	reqBody["secrets"] = secrets
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid secrets"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/secrets", params, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to set secrets", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	computed := map[string]models.ComputedSecret{}
	for key, secret := range result["secrets"].(map[string]interface{}) {
		val, ok := secret.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type mismatch for secret, expected map[string]interface{}, got %T", secret), Message: "Unable to parse API response", Code: statusCode}
		}
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return computed, Error{}
}

// GetSecretNames for specified project and config
func GetSecretNames(host string, verifyTLS bool, apiKey string, project string, config string, includeDynamicSecrets bool) ([]string, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "include_dynamic_secrets", Value: strconv.FormatBool(includeDynamicSecrets)})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/secrets/names", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch secret names", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	namesInterface, ok := result["names"].([]interface{})
	if !ok {
		return nil, Error{Err: fmt.Errorf("Unexpected type parsing secret names, expected []interface{}, got %T", result["names"]), Message: "Unable to parse API response", Code: statusCode}
	}

	var names []string
	for _, value := range namesInterface {
		v, ok := value.(string)
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type parsing secret name, expected string, got %T", value), Message: "Unable to parse API response", Code: statusCode}
		}
		names = append(names, v)
	}

	return names, Error{}
}

// UploadSecrets for specified project and config
func UploadSecrets(host string, verifyTLS bool, apiKey string, project string, config string, secrets string) (map[string]models.ComputedSecret, Error) {
	reqBody := map[string]interface{}{}
	reqBody["file"] = secrets
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid file"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/secrets/upload", params, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to upload secrets", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	computed := map[string]models.ComputedSecret{}
	for key, secret := range result["secrets"].(map[string]interface{}) {
		val, ok := secret.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for secret, expected map[string]interface{}, got %T", secret), Message: "Unable to parse API response", Code: statusCode}
		}
		computed[key] = models.ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return computed, Error{}
}

// GetWorkplaceSettings get specified workplace settings
func GetWorkplaceSettings(host string, verifyTLS bool, apiKey string) (models.WorkplaceSettings, Error) {
	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/workplace", []queryParam{})
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to fetch workplace settings", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	workplace, ok := result["workplace"].(map[string]interface{})
	if !ok {
		return models.WorkplaceSettings{}, Error{Err: fmt.Errorf("Unexpected type parsing WorkplaceSettings, expected map[string]interface{}, got %T", result["workplace"]), Message: "Unable to parse API response", Code: statusCode}
	}
	settings := models.ParseWorkplaceSettings(workplace)
	return settings, Error{}
}

// SetWorkplaceSettings set workplace settings
func SetWorkplaceSettings(host string, verifyTLS bool, apiKey string, values models.WorkplaceSettings) (models.WorkplaceSettings, Error) {
	body, err := json.Marshal(values)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Invalid workplace settings"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/workplace", []queryParam{}, body)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to update workplace settings", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.WorkplaceSettings{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	workplace, ok := result["workplace"].(map[string]interface{})
	if !ok {
		return models.WorkplaceSettings{}, Error{Err: fmt.Errorf("Unexpected type parsing WorkplaceSettings, expected map[string]interface{}, got %T", result["workplace"]), Message: "Unable to parse API response", Code: statusCode}
	}
	settings := models.ParseWorkplaceSettings(workplace)
	return settings, Error{}
}

// GetProjects get projects
func GetProjects(host string, verifyTLS bool, apiKey string) ([]models.ProjectInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "per_page", Value: "100"})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/projects", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch projects", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var info []models.ProjectInfo
	for _, project := range result["projects"].([]interface{}) {
		project, ok := project.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for project, expected map[string]interface{}, got %T", project), Message: "Unable to parse API response", Code: statusCode}
		}
		projectInfo := models.ParseProjectInfo(project)
		info = append(info, projectInfo)
	}
	return info, Error{}
}

// GetProject get specified project
func GetProject(host string, verifyTLS bool, apiKey string, project string) (models.ProjectInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/projects/project", params)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to fetch project", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	resultProject, ok := result["project"].(map[string]interface{})
	if !ok {
		return models.ProjectInfo{}, Error{Err: fmt.Errorf("Unexpected type for project, expected map[string]interface{}, got %T", result["project"]), Message: "Unable to parse API response", Code: statusCode}
	}
	projectInfo := models.ParseProjectInfo(resultProject)
	return projectInfo, Error{}
}

// CreateProject create a project
func CreateProject(host string, verifyTLS bool, apiKey string, name string, description string) (models.ProjectInfo, Error) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Invalid project info"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/projects", []queryParam{}, body)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to create project", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	resultProject, ok := result["project"].(map[string]interface{})
	if !ok {
		return models.ProjectInfo{}, Error{Err: fmt.Errorf("Unexpected type for project, expected map[string]interface{}, got %T", result["project"]), Message: "Unable to parse API response", Code: statusCode}
	}
	projectInfo := models.ParseProjectInfo(resultProject)
	return projectInfo, Error{}
}

// UpdateProject update a project's name and (optional) description
func UpdateProject(host string, verifyTLS bool, apiKey string, project string, name string, description ...string) (models.ProjectInfo, Error) {
	postBody := map[string]string{"name": name}
	if len(description) > 0 {
		desc := description[0]
		postBody["description"] = desc
	}

	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Invalid project info"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/projects/project", params, body)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to update project", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ProjectInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	resultProject, ok := result["project"].(map[string]interface{})
	if !ok {
		return models.ProjectInfo{}, Error{Err: fmt.Errorf("Unexpected type for project, expected map[string]interface{}, got %T", result["project"]), Message: "Unable to parse API response", Code: statusCode}
	}
	projectInfo := models.ParseProjectInfo(resultProject)
	return projectInfo, Error{}
}

// DeleteProject delete a project
func DeleteProject(host string, verifyTLS bool, apiKey string, project string) Error {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})

	statusCode, _, response, err := DeleteRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/projects/project", params)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete project", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return Error{}
}

// GetEnvironments get environments
func GetEnvironments(host string, verifyTLS bool, apiKey string, project string) ([]models.EnvironmentInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/environments", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch environments", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var info []models.EnvironmentInfo
	for _, environment := range result["environments"].([]interface{}) {
		environment, ok := environment.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for environment, expected map[string]interface{}, got %T", environment), Message: "Unable to parse API response", Code: statusCode}
		}
		environmentInfo := models.ParseEnvironmentInfo(environment)
		info = append(info, environmentInfo)
	}
	return info, Error{}
}

// GetEnvironment get specified environment
func GetEnvironment(host string, verifyTLS bool, apiKey string, project string, environment string) (models.EnvironmentInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "environment", Value: environment})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/environments/environment", params)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to fetch environment", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	environmentInfo, ok := result["environment"].(map[string]interface{})
	if !ok {
		return models.EnvironmentInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing environment, expected map[string]interface{}, got %T", result["environment"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseEnvironmentInfo(environmentInfo)
	return info, Error{}
}

// CreateEnvironment create an environment
func CreateEnvironment(host string, verifyTLS bool, apiKey string, project string, name string, slug string) (models.EnvironmentInfo, Error) {
	postBody := map[string]string{"project": project, "name": name, "slug": slug}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Invalid environment info"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/environments", []queryParam{}, body)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to create environment", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	environmentInfo, ok := result["environment"].(map[string]interface{})
	if !ok {
		return models.EnvironmentInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing environment, expected map[string]interface{}, got %T", result["environment"]), Message: "Unable to parse API response", Code: statusCode}
	}

	info := models.ParseEnvironmentInfo(environmentInfo)

	return info, Error{}
}

// DeleteEnvironment delete an environment
func DeleteEnvironment(host string, verifyTLS bool, apiKey string, project string, environment string) Error {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "environment", Value: environment})

	statusCode, _, response, err := DeleteRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/environments/environment", params)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete environment", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return Error{}
}

// RenameEnvironment rename an environment
func RenameEnvironment(host string, verifyTLS bool, apiKey string, project string, environment string, name string, slug string) (models.EnvironmentInfo, Error) {
	postBody := map[string]string{"project": project, "environment": environment}
	if name != "" {
		postBody["name"] = name
	}
	if slug != "" {
		postBody["slug"] = slug
	}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Invalid environment info"}
	}

	statusCode, _, response, err := PutRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/environments/environment", []queryParam{}, body)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to rename environment", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.EnvironmentInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	environmentInfo, ok := result["environment"].(map[string]interface{})
	if !ok {
		return models.EnvironmentInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing environment, expected map[string]interface{}, got %T", result["environment"]), Message: "Unable to parse API response", Code: statusCode}
	}

	info := models.ParseEnvironmentInfo(environmentInfo)
	return info, Error{}
}

// GetConfigs get configs
func GetConfigs(host string, verifyTLS bool, apiKey string, project string) ([]models.ConfigInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "per_page", Value: "100"})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch configs", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var info []models.ConfigInfo
	for _, config := range result["configs"].([]interface{}) {
		config, ok := config.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type parsing config, expected map[string]interface{}, got %T", config), Message: "Unable to parse API response", Code: statusCode}
		}
		configInfo := models.ParseConfigInfo(config)
		info = append(info, configInfo)
	}
	return info, Error{}
}

// GetConfig get a config
func GetConfig(host string, verifyTLS bool, apiKey string, project string, config string) (models.ConfigInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config", params)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to fetch configs", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	configInfo, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(configInfo)
	return info, Error{}
}

// CreateConfig create a config
func CreateConfig(host string, verifyTLS bool, apiKey string, project string, name string, environment string) (models.ConfigInfo, Error) {
	postBody := map[string]interface{}{"name": name, "environment": environment}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Invalid config info"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs", params, body)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to create config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	config, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(config)
	return info, Error{}
}

// DeleteConfig delete a config
func DeleteConfig(host string, verifyTLS bool, apiKey string, project string, config string) Error {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := DeleteRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config", params)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return Error{}
}

// LockConfig lock a config
func LockConfig(host string, verifyTLS bool, apiKey string, project string, config string) (models.ConfigInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/lock", params, nil)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to lock config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	configInfo, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config info, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(configInfo)
	return info, Error{}
}

// UnlockConfig unlock a config
func UnlockConfig(host string, verifyTLS bool, apiKey string, project string, config string) (models.ConfigInfo, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/unlock", params, nil)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to unlock config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	configInfo, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config info, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(configInfo)
	return info, Error{}
}

// CloneConfig clone a config
func CloneConfig(host string, verifyTLS bool, apiKey string, project string, config string, name string) (models.ConfigInfo, Error) {
	postBody := map[string]interface{}{"name": name}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Invalid config info"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/clone", params, body)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to clone config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	configInfo, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config info, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(configInfo)
	return info, Error{}
}

// UpdateConfig update a config
func UpdateConfig(host string, verifyTLS bool, apiKey string, project string, config string, name string) (models.ConfigInfo, Error) {
	postBody := map[string]interface{}{"name": name}
	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Invalid config info"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config", params, body)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to update config", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigInfo{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	configInfo, ok := result["config"].(map[string]interface{})
	if !ok {
		return models.ConfigInfo{}, Error{Err: fmt.Errorf("Unexpected type parsing config info, expected map[string]interface{}, got %T", result["config"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigInfo(configInfo)
	return info, Error{}
}

// GetActivityLogs get activity logs
func GetActivityLogs(host string, verifyTLS bool, apiKey string, page int, number int) ([]models.ActivityLog, Error) {
	var params []queryParam
	if page != 0 {
		params = append(params, queryParam{Key: "page", Value: fmt.Sprint(page)})
	}
	if number != 0 {
		params = append(params, queryParam{Key: "per_page", Value: fmt.Sprint(number)})
	}
	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/logs", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch activity logs", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var logs []models.ActivityLog
	for _, log := range result["logs"].([]interface{}) {
		log, ok := log.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type parsing activity log, expected map[string]interface{}, got %T", log), Message: "Unable to parse API response", Code: statusCode}
		}
		parsedLog := models.ParseActivityLog(log)
		logs = append(logs, parsedLog)
	}
	return logs, Error{}
}

// GetActivityLog get specified activity log
func GetActivityLog(host string, verifyTLS bool, apiKey string, log string) (models.ActivityLog, Error) {
	params := []queryParam{{Key: "log", Value: log}}
	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/logs/log", params)
	if err != nil {
		return models.ActivityLog{}, Error{Err: err, Message: "Unable to fetch activity log", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ActivityLog{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	logResult, ok := result["log"].(map[string]interface{})
	if !ok {
		return models.ActivityLog{}, Error{Err: fmt.Errorf("Unexpected type for log, expected map[string]interface{}, got %T", result["log"]), Message: "Unable to parse API response", Code: statusCode}
	}
	parsedLog := models.ParseActivityLog(logResult)
	return parsedLog, Error{}
}

// GetConfigLogs get config audit logs
func GetConfigLogs(host string, verifyTLS bool, apiKey string, project string, config string, page int, number int) ([]models.ConfigLog, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	if page != 0 {
		params = append(params, queryParam{Key: "page", Value: fmt.Sprint(page)})
	}
	if number != 0 {
		params = append(params, queryParam{Key: "per_page", Value: fmt.Sprint(number)})
	}

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/logs", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch config logs", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var logs []models.ConfigLog
	for _, log := range result["logs"].([]interface{}) {
		log, ok := log.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for ConfigLog response, expected map[string]interface{}, got %T", log), Message: "Unable to parse API response", Code: statusCode}
		}
		parsedLog := models.ParseConfigLog(log)
		logs = append(logs, parsedLog)
	}
	return logs, Error{}
}

// GetConfigLog get config audit log
func GetConfigLog(host string, verifyTLS bool, apiKey string, project string, config string, log string) (models.ConfigLog, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "log", Value: log})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/logs/log", params)
	if err != nil {
		return models.ConfigLog{}, Error{Err: err, Message: "Unable to fetch config log", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigLog{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	logResult, ok := result["log"].(map[string]interface{})
	if !ok {
		return models.ConfigLog{}, Error{Err: fmt.Errorf("Unexpected type parsing ConfigLog result, expected map[string]interface{}, got %T", result["log"]), Message: "Unable to parse API response", Code: statusCode}
	}
	parsedLog := models.ParseConfigLog(logResult)
	return parsedLog, Error{}
}

// RollbackConfigLog rollback a config log
func RollbackConfigLog(host string, verifyTLS bool, apiKey string, project string, config string, log string) (models.ConfigLog, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "log", Value: log})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/logs/log/rollback", params, nil)
	if err != nil {
		return models.ConfigLog{}, Error{Err: err, Message: "Unable to rollback config log", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigLog{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	logResult, ok := result["log"].(map[string]interface{})
	if !ok {
		return models.ConfigLog{}, Error{Err: fmt.Errorf("Unexpected type for ConfigLog response, expected map[string]interface{}, got %T", result["log"]), Message: "Unable to parse API response", Code: statusCode}
	}
	parsedLog := models.ParseConfigLog(logResult)
	return parsedLog, Error{}
}

// GetConfigServiceTokens get config service tokens
func GetConfigServiceTokens(host string, verifyTLS bool, apiKey string, project string, config string) ([]models.ConfigServiceToken, Error) {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := GetRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/tokens", params)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch service tokens", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var tokens []models.ConfigServiceToken
	for _, token := range result["tokens"].([]interface{}) {
		token, ok := token.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for ConfigServiceToken response, expected map[string]interface{}, got %T", token), Message: "Unable to parse API response", Code: statusCode}
		}
		parsedToken := models.ParseConfigServiceToken(token)
		tokens = append(tokens, parsedToken)
	}
	return tokens, Error{}
}

// CreateConfigServiceToken create a config service token
func CreateConfigServiceToken(host string, verifyTLS bool, apiKey string, project string, config string, name string, expireAt time.Time, access string) (models.ConfigServiceToken, Error) {
	postBody := map[string]interface{}{"name": name}
	if !expireAt.IsZero() {
		postBody["expire_at"] = expireAt.Unix()
	}
	postBody["access"] = access

	body, err := json.Marshal(postBody)
	if err != nil {
		return models.ConfigServiceToken{}, Error{Err: err, Message: "Invalid service token info"}
	}

	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/tokens", params, body)
	if err != nil {
		return models.ConfigServiceToken{}, Error{Err: err, Message: "Unable to create service token", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return models.ConfigServiceToken{}, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	tokenResult, ok := result["token"].(map[string]interface{})
	if !ok {
		return models.ConfigServiceToken{}, Error{Err: fmt.Errorf("Unexpected type for token result in ConfigServiceToken, expected map[string]interface{}, got %T", result["token"]), Message: "Unable to parse API response", Code: statusCode}
	}
	info := models.ParseConfigServiceToken(tokenResult)
	return info, Error{}
}

// DeleteConfigServiceToken delete a config service token
func DeleteConfigServiceToken(host string, verifyTLS bool, apiKey string, project string, config string, slug string) Error {
	var params []queryParam
	params = append(params, queryParam{Key: "project", Value: project})
	params = append(params, queryParam{Key: "config", Value: config})
	params = append(params, queryParam{Key: "slug", Value: slug})

	statusCode, _, response, err := DeleteRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/configs/config/tokens/token", params)
	if err != nil {
		return Error{Err: err, Message: "Unable to delete service token", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	return Error{}
}

// ImportTemplate import projects from a template file
func ImportTemplate(host string, verifyTLS bool, apiKey string, template []byte) ([]models.ProjectInfo, Error) {
	reqBody := map[string]interface{}{}
	reqBody["template"] = string(template)
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Invalid template"}
	}

	statusCode, _, response, err := PostRequest(host, verifyTLS, apiKeyHeader(apiKey), "/v3/workplace/template/import", []queryParam{}, body)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to import project(s)", Code: statusCode}
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse API response", Code: statusCode}
	}

	var info []models.ProjectInfo
	for _, project := range result["projects"].([]interface{}) {
		project, ok := project.(map[string]interface{})
		if !ok {
			return nil, Error{Err: fmt.Errorf("Unexpected type for project, expected map[string]interface{}, got %T", project), Message: "Unable to parse API response", Code: statusCode}
		}
		projectInfo := models.ParseProjectInfo(project)
		info = append(info, projectInfo)
	}
	return info, Error{}
}
