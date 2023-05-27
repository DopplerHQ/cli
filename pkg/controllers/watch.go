/*
Copyright © 2023 Doppler <support@doppler.com>

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
	"encoding/json"
	"strings"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

func ParseWatchEvent(data []byte) models.WatchSecrets {
	// Expected format: "event: message\ndata: {JSON}\n\n"
	var parts []string
	for _, s := range strings.Split(string(data), "\n") {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if len(parts) != 2 {
		utils.LogDebug("Unable to parse API response; invalid length")
		CaptureEvent("WatchDataParseError", map[string]interface{}{"error": "invalid length"})
		return models.WatchSecrets{}
	}
	if parts[0] != "event: message" {
		utils.LogDebug("Unable to parse API response; invalid event")
		CaptureEvent("WatchDataParseError", map[string]interface{}{"error": "invalid event"})
		return models.WatchSecrets{}
	}
	const dataPrefix = "data: "
	if !strings.HasPrefix(parts[1], dataPrefix) {
		utils.LogDebug("Unable to parse API response; invalid data")
		CaptureEvent("WatchDataParseError", map[string]interface{}{"error": "invalid data"})
		return models.WatchSecrets{}
	}

	dataJson := strings.TrimPrefix(parts[1], dataPrefix)
	var watchSecrets models.WatchSecrets
	err := json.Unmarshal([]byte(dataJson), &watchSecrets)
	if err != nil {
		CaptureEvent("WatchDataParseError", map[string]interface{}{"error": "invalid data json"})
		utils.LogDebug("Unable to parse API response")
		utils.LogDebugError(err)
		return models.WatchSecrets{}
	}

	return watchSecrets
}
