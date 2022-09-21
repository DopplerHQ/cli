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
package utils

import (
	"testing"
)

func TestIsDotNETSettingsFile(t *testing.T) {
	type testCase struct {
		fileName string
		match    bool
	}

	testCases := []testCase{
		{"appSettings.json", true},
		{"appsettings.json", true},
		{"appsettings.Development.json", true},
		{"appSettings.Production.json", true},
		{"appSettings.Other_Environment.json", true},
		{"settings.json", false},
		{"app.settings.json", false},
	}

	for _, testCase := range testCases {
		matched := IsDotNETSettingsFile(testCase.fileName)
		if testCase.match != matched {
			t.Errorf("Expected '%s' match result to be '%t' but got '%t'", testCase.fileName, testCase.match, matched)
		}
	}
}
