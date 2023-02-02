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
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

func GetConfigs(config models.ScopedOptions) ([]models.ConfigInfo, Error) {
	utils.RequireValue("token", config.Token.Value)

	configs, err := http.GetConfigs(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, "", 1, 100)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	return configs, Error{}
}

func GetConfigNames(config models.ScopedOptions) ([]string, Error) {
	configs, err := GetConfigs(config)
	if !err.IsNil() {
		return nil, err
	}

	var names []string
	for _, config := range configs {
		names = append(names, config.Name)
	}
	return names, Error{}
}

func GetConfigLogIDs(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	logs, err := http.GetConfigLogs(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, 0, 0)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	var names []string
	for _, log := range logs {
		names = append(names, log.ID)
	}
	return names, Error{}
}

func GetConfigTokenSlugs(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	tokens, err := http.GetConfigServiceTokens(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	var slugs []string
	for _, token := range tokens {
		slugs = append(slugs, token.Slug)
	}
	return slugs, Error{}
}

func GetEnvironmentIDs(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	environments, err := http.GetEnvironments(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, 1, 100)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	var ids []string
	for _, environment := range environments {
		ids = append(ids, environment.ID)
	}
	return ids, Error{}
}
