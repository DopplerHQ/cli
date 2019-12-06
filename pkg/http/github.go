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
	"fmt"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
)

func getLatestVersion() (string, error) {
	origTimeout := TimeoutDuration
	TimeoutDuration = 2 * time.Second
	_, resp, err := GetRequest("https://api.github.com", true, nil, "/repos/DopplerHQ/cli/releases/latest", nil)
	TimeoutDuration = origTimeout
	if err != nil {
		return "", err
	}

	var body map[string]interface{}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return "", err
	}

	if version, exists := body["tag_name"]; exists {
		return version.(string), nil
	}

	return "", errors.New("unable to retrieve tag_name of latest release")
}

// CheckCLIVersion check for updates to the CLI
func CheckCLIVersion(versionCheck models.VersionCheck, silent bool, json bool, debug bool) models.VersionCheck {
	now := time.Now()
	// only check version if more than a day since last check
	if !now.After(versionCheck.CheckedAt.Add(24 * time.Hour)) {
		return models.VersionCheck{}
	}

	tag, err := getLatestVersion()
	if err != nil {
		if debug && !json {
			fmt.Println("Error:", err)
		}
		if !silent && !json {
			fmt.Println("Unable to check for CLI updates")
		}
		return models.VersionCheck{}
	}

	versionCheck.CheckedAt = now
	if versionCheck.LatestVersion != tag {
		versionCheck.LatestVersion = tag
	}

	return versionCheck
}
