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
package models

import (
	"encoding/json"
)

// ParseWorkplaceSettings parse workplace settings
func ParseWorkplaceSettings(info map[string]interface{}) WorkplaceSettings {
	var workplaceInfo WorkplaceSettings

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

// ParseProjectInfo parse project info
func ParseProjectInfo(info map[string]interface{}) ProjectInfo {
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

	return projectInfo
}

// ParseEnvironmentInfo parse environment info
func ParseEnvironmentInfo(info map[string]interface{}) EnvironmentInfo {
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

	return environmentInfo
}

// ParseConfigInfo parse config info
func ParseConfigInfo(info map[string]interface{}) ConfigInfo {
	var configInfo ConfigInfo

	if info["name"] != nil {
		configInfo.Name = info["name"].(string)
	}
	if info["stage"] != nil {
		configInfo.Environment = info["stage"].(string)
	}
	if info["pipeline"] != nil {
		configInfo.Project = info["pipeline"].(string)
	}
	if info["created_at"] != nil {
		configInfo.CreatedAt = info["created_at"].(string)
	}
	if info["deployed_at"] != nil {
		configInfo.DeployedAt = info["deployed_at"].(string)
	}

	return configInfo
}

// ParseConfigLog parse config log
func ParseConfigLog(log map[string]interface{}) ConfigLog {
	var parsedLog ConfigLog

	if log["id"] != nil {
		parsedLog.ID = log["id"].(string)
	}
	if log["text"] != nil {
		parsedLog.Text = log["text"].(string)
	}
	if log["html"] != nil {
		parsedLog.HTML = log["html"].(string)
	}
	if log["created_at"] != nil {
		parsedLog.CreatedAt = log["created_at"].(string)
	}
	if log["environment"] != nil {
		parsedLog.Config = log["environment"].(string)
	}
	if log["stage"] != nil {
		parsedLog.Environment = log["stage"].(string)
	}
	if log["pipeline"] != nil {
		parsedLog.Project = log["pipeline"].(string)
	}
	if log["user"] != nil {
		user := log["user"].(map[string]interface{})
		parsedLog.User.Email = user["email"].(string)
		parsedLog.User.Name = user["name"].(string)
		parsedLog.User.Username = user["username"].(string)
		parsedLog.User.ProfileImage = user["profile_image_url"].(string)
	}
	if log["diff"] != nil {
		for _, diff := range log["diff"].([]interface{}) {
			diffMap := diff.(map[string]interface{})
			d := LogDiff{}
			if diffMap["name"] != nil {
				d.Name = diffMap["name"].(string)
			}
			if diffMap["added"] != nil {
				d.Added = diffMap["added"].(string)
			}
			if diffMap["removed"] != nil {
				d.Removed = diffMap["removed"].(string)
			}
			parsedLog.Diff = append(parsedLog.Diff, d)
		}
	}

	return parsedLog
}

// ParseSecrets for specified project and config
func ParseSecrets(response []byte) (map[string]ComputedSecret, error) {
	var result map[string]interface{}
	err := json.Unmarshal(response, &result)
	if err != nil {
		return nil, err
	}

	computed := map[string]ComputedSecret{}
	secrets := result["secrets"].(map[string]interface{})
	for key, secret := range secrets {
		val := secret.(map[string]interface{})
		computed[key] = ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
	}

	return computed, nil
}
