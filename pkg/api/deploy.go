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
	"strconv"

	"github.com/DopplerHQ/cli/pkg/utils"
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
