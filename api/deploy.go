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
	utils "cli/utils"
	"encoding/json"
	"strconv"

	"github.com/spf13/cobra"
)

type errorResponse struct {
	messages []string
	success  bool
}

// Secret key/value
type Secret struct {
	Name  string
	Value string
}

// ParseDeploySecrets parse deploy secrets
func ParseDeploySecrets(response []byte) (map[string]string, error) {
	var result map[string]interface{}
	err := json.Unmarshal(response, &result)
	if err != nil {
		return nil, err
	}

	parsedSecrets := make(map[string]string)
	secrets := result["variables"].(map[string]interface{})
	for name, value := range secrets {
		parsedSecrets[name] = value.(string)
	}

	return parsedSecrets, nil
}

// GetDeploySecrets for specified project and config
func GetDeploySecrets(cmd *cobra.Command, host string, apiKey string, project string, config string) ([]byte, error) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	response, err := utils.GetRequest(host, "/v1/variables", params, apiKey)
	if err != nil {
		utils.Log("Unable to fetch secrets")
		return nil, err
	}

	return response, nil
}

// DownloadSecrets for specified project and config
func DownloadSecrets(cmd *cobra.Command, host string, apiKey string, project string, config string, metadata bool) []byte {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})
	params = append(params, utils.QueryParam{Key: "format", Value: "file"})
	params = append(params, utils.QueryParam{Key: "metadata", Value: strconv.FormatBool(metadata)})

	response, err := utils.GetRequest(host, "/v1/variables", params, apiKey)
	if err != nil {
		utils.Err(err, "Unable to download secrets")
	}

	return response
}
