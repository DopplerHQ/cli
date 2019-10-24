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
	utils "doppler-cli/utils"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

// ComputedSecret holds computed and raw value
type ComputedSecret struct {
	Name          string `json:"name"`
	RawValue      string `json:"raw"`
	ComputedValue string `json:"computed"`
}

// WorkplaceInfo workplace info
type WorkplaceInfo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	BillingEmail string `json:"billing_email"`
}

// ProjectInfo project info
type ProjectInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	SetupAt     string `json:"setup_at"`
}

// EnvironmentInfo environment info
type EnvironmentInfo struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	CreatedAt        string   `json:"created_at"`
	FirstDeployAt    string   `json:"first_deploy_at"`
	SetupAt          string   `json:"setup_at"`
	Project          string   `json:"pipeline"`
	MissingVariables []string `json:"missing_variables"`
}

func parseWorkplaceInfo(info map[string]interface{}) WorkplaceInfo {
	var workplaceInfo WorkplaceInfo

	if info["id"] != nil {
		workplaceInfo.ID = info["id"].(string)
	}
	if info["name"] != nil {
		workplaceInfo.Name = info["name"].(string)
	}
	if info["billing_email"] != nil {
		workplaceInfo.BillingEmail = info["billing_email"].(string)
	}

	return workplaceInfo
}

func parseProjectInfo(info map[string]interface{}) ProjectInfo {
	var projectInfo ProjectInfo

	if info["id"] != nil {
		projectInfo.ID = info["id"].(string)
	}
	if info["name"] != nil {
		projectInfo.Name = info["name"].(string)
	}
	if info["description"] != nil {
		projectInfo.Description = info["description"].(string)
	}
	if info["created_at"] != nil {
		projectInfo.CreatedAt = info["created_at"].(string)
	}
	if info["setup_at"] != nil {
		projectInfo.SetupAt = info["setup_at"].(string)
	}

	return projectInfo
}

func parseEnvironmentInfo(info map[string]interface{}) EnvironmentInfo {
	var environmentInfo EnvironmentInfo

	if info["id"] != nil {
		environmentInfo.ID = info["id"].(string)
	}
	if info["name"] != nil {
		environmentInfo.Name = info["name"].(string)
	}
	if info["created_at"] != nil {
		environmentInfo.CreatedAt = info["created_at"].(string)
	}
	if info["first_deploy_at"] != nil {
		environmentInfo.FirstDeployAt = info["first_deploy_at"].(string)
	}
	if info["setup_at"] != nil {
		environmentInfo.SetupAt = info["setup_at"].(string)
	}
	if info["pipeline"] != nil {
		environmentInfo.Project = info["pipeline"].(string)
	}
	if info["missing_variables"] != nil {
		var missingVariables []string
		for _, val := range info["missing_variables"].([]interface{}) {
			missingVariables = append(missingVariables, val.(string))
		}
		environmentInfo.MissingVariables = missingVariables
	}

	return environmentInfo
}

// GetAPISecrets for specified project and config
func GetAPISecrets(cmd *cobra.Command, apiKey string, project string, config string) ([]byte, map[string]ComputedSecret) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/variables", params, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch secrets")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	computed := make(map[string]ComputedSecret)
	secrets := result["variables"].(map[string]interface{})
	// fmt.Println("secret1", secrets)
	for key, secret := range secrets {
		val := secret.(map[string]interface{})
		computed[key] = ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return response, computed
}

// SetAPISecrets for specified project and config
func SetAPISecrets(cmd *cobra.Command, apiKey string, project string, config string, secrets map[string]interface{}) ([]byte, map[string]ComputedSecret) {
	reqBody := make(map[string]interface{})
	reqBody["variables"] = secrets
	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Println("Invalid secrets")
		utils.Err(err)
	}

	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/variables", params, apiKey, body)
	if err != nil {
		fmt.Println("Unable to fetch secrets")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	computed := make(map[string]ComputedSecret)
	for key, secret := range result["variables"].(map[string]interface{}) {
		val := secret.(map[string]interface{})
		computed[key] = ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return response, computed
}

// GetAPIWorkplace get workplace info
func GetAPIWorkplace(cmd *cobra.Command, apiKey string) ([]byte, WorkplaceInfo) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/workplace", []utils.QueryParam{}, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch workplace")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	info := parseWorkplaceInfo(result["workplace"].(map[string]interface{}))
	return response, info
}

// SetAPIWorkplace set workplace info
func SetAPIWorkplace(cmd *cobra.Command, apiKey string, values WorkplaceInfo) ([]byte, WorkplaceInfo) {
	body, err := json.Marshal(values)
	if err != nil {
		fmt.Println("Invalid workplace info")
		utils.Err(err)
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/workplace", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		fmt.Println("Unable to update workplace info")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	info := parseWorkplaceInfo(result["workplace"].(map[string]interface{}))
	return response, info
}

// GetAPIProjects get projects
func GetAPIProjects(cmd *cobra.Command, apiKey string) ([]byte, []ProjectInfo) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/pipelines", []utils.QueryParam{}, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch projects")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	var info []ProjectInfo
	for _, project := range result["pipelines"].([]interface{}) {
		projectInfo := parseProjectInfo(project.(map[string]interface{}))
		info = append(info, projectInfo)
	}
	return response, info
}

// GetAPIProject get project
func GetAPIProject(cmd *cobra.Command, apiKey string, project string) ([]byte, ProjectInfo) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch project")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	projectInfo := parseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// CreateAPIProject create a project
func CreateAPIProject(cmd *cobra.Command, apiKey string, name string, description string) ([]byte, ProjectInfo) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		fmt.Println("Invalid project info")
		utils.Err(err)
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/pipelines/", []utils.QueryParam{}, apiKey, body)
	if err != nil {
		fmt.Println("Unable to create project")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	projectInfo := parseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// UpdateAPIProject update a project
func UpdateAPIProject(cmd *cobra.Command, apiKey string, project string, name string, description string) ([]byte, ProjectInfo) {
	postBody := map[string]string{"name": name, "description": description}
	body, err := json.Marshal(postBody)
	if err != nil {
		fmt.Println("Invalid project info")
		utils.Err(err)
	}

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.PostRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey, body)
	if err != nil {
		fmt.Println("Unable to update project")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	projectInfo := parseProjectInfo(result["pipeline"].(map[string]interface{}))
	return response, projectInfo
}

// DeleteAPIProject create a project
func DeleteAPIProject(cmd *cobra.Command, apiKey string, project string) {
	host := cmd.Flag("api-host").Value.String()
	response, err := utils.DeleteRequest(host, "v2/pipelines/"+project, []utils.QueryParam{}, apiKey)
	if err != nil {
		fmt.Println("Unable to delete project")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}
}

// GetAPIEnvironments get environments
func GetAPIEnvironments(cmd *cobra.Command, apiKey string, project string) ([]byte, []EnvironmentInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/stages", params, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch environments")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	var info []EnvironmentInfo
	for _, environment := range result["stages"].([]interface{}) {
		environmentInfo := parseEnvironmentInfo(environment.(map[string]interface{}))
		info = append(info, environmentInfo)
	}
	return response, info
}

// GetAPIEnvironment get specified environment
func GetAPIEnvironment(cmd *cobra.Command, apiKey string, project string, environment string) ([]byte, EnvironmentInfo) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/stages/"+environment, params, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch environment")
		utils.Err(err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(response, &result)
	if err != nil {
		utils.Err(err)
	}

	info := parseEnvironmentInfo(result["stage"].(map[string]interface{}))
	return response, info
}
