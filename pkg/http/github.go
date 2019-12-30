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
	"os"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
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
	utils.LogDebug("Checking for latest version of the CLI")
	tag, err := getLatestVersion()
	if err != nil {
		if debug && !json {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		if !silent && !json {
			fmt.Fprintln(os.Stderr, "Unable to check for CLI updates")
		}
		return models.VersionCheck{}
	}

	versionCheck.CheckedAt = time.Now()
	tag = normalizeVersion(tag)
	if normalizeVersion(versionCheck.LatestVersion) != tag {
		versionCheck.LatestVersion = tag
	}

	return versionCheck
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}

	return version
}
