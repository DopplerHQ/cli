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
	"strings"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/global"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/utils"
)

// This package collects anonymous analytics for the purpose of improving the Doppler CLI

func CaptureCommand(command string) {
	global.WaitGroup.Add(1)
	go captureCommand(command)
}

func captureCommand(command string) {
	defer global.WaitGroup.Done()

	if !configuration.IsAnalyticsEnabled() {
		return
	}

	command = strings.ReplaceAll(command, " ", ".")
	if _, err := http.CaptureCommand(command); !err.IsNil() {
		utils.LogDebugError(err.Unwrap())
	}
}

func CaptureEvent(event string, metadata map[string]interface{}) {
	global.WaitGroup.Add(1)
	go captureEvent(event, metadata)
}

func captureEvent(event string, metadata map[string]interface{}) {
	defer global.WaitGroup.Done()

	if !configuration.IsAnalyticsEnabled() {
		return
	}

	if _, err := http.CaptureEvent(event, metadata); !err.IsNil() {
		utils.LogDebugError(err.Unwrap())
	}
}
