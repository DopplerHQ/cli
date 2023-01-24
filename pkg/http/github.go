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
	"encoding/json"
	"errors"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
)

const cliHostname = "https://cli.doppler.com"

func getLatestVersion() (string, error) {
	origTimeout := TimeoutDuration
	TimeoutDuration = 2 * time.Second

	url, err := generateURL(cliHostname, "/version", nil)
	if err != nil {
		return "", err
	}

	_, _, resp, err := GetRequest(url, true, nil)
	TimeoutDuration = origTimeout
	if err != nil {
		return "", err
	}

	var body map[string]interface{}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return "", err
	}

	if version, exists := body["version"]; exists {
		return version.(string), nil
	}

	return "", errors.New("unable to retrieve latest CLI version")
}

// GetLatestCLIVersion fetches the latest CLI version
func GetLatestCLIVersion() (models.VersionCheck, error) {
	utils.LogDebug("Checking for latest version of the CLI")
	tag, err := getLatestVersion()
	if err != nil {
		utils.LogDebug("Unable to check for CLI updates")
		utils.LogDebugError(err)
		return models.VersionCheck{}, err
	}

	versionCheck := models.VersionCheck{CheckedAt: time.Now(), LatestVersion: version.Normalize(tag)}
	return versionCheck, nil
}

// GetCLIInstallScript from cli.doppler.com
func GetCLIInstallScript() ([]byte, Error) {
	url, err := generateURL(cliHostname, "/install.sh", nil)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to generate url"}
	}

	_, _, resp, err := GetRequest(url, true, nil)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to download CLI install script"}
	}
	return resp, Error{}
}

// GetChangelog of CLI releases
func GetChangelog() ([]byte, Error) {
	url, err := generateURL(cliHostname, "/changes", nil)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to generate url"}
	}

	headers := map[string]string{"Accept": "application/json"}
	_, _, resp, err := GetRequest(url, true, headers)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to fetch changelog"}
	}
	return resp, Error{}
}
