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

// GetAPISecrets for specified project and config
func GetAPISecrets(cmd *cobra.Command, apiKey string, project string, config string, parse bool) ([]byte, map[string]ComputedSecret) {
	var params []utils.QueryParam
	params = append(params, utils.QueryParam{Key: "environment", Value: config})
	params = append(params, utils.QueryParam{Key: "pipeline", Value: project})

	host := cmd.Flag("api-host").Value.String()
	response, err := utils.GetRequest(host, "v2/variables", params, apiKey)
	if err != nil {
		fmt.Println("Unable to fetch secrets")
		utils.Err(err)
		return nil, nil
	}

	if parse {
		var result map[string]interface{}
		err = json.Unmarshal(response, &result)
		if err != nil {
			utils.Err(err)
		}

		computed := make(map[string]ComputedSecret)
		secrets := result["variables"].(map[string]interface{})
		for key, secret := range secrets {
			val := secret.(map[string]interface{})
			computed[key] = ComputedSecret{Name: key, RawValue: val["raw"].(string), ComputedValue: val["computed"].(string)}
		}

		return response, computed
	}

	return response, nil
}
