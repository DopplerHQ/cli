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
package controllers

import (
	"sort"

	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

func GetSecretNames(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	response, err := http.GetSecrets(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		return nil, Error{Err: parseErr, Message: "Unable to parse API response"}
	}

	var secretsNames []string
	for name := range secrets {
		secretsNames = append(secretsNames, name)
	}
	sort.Strings(secretsNames)

	return secretsNames, Error{}
}
