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

package controllers

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type dangerousSecretNameTestCase struct {
	name                         string
	secrets                      map[string]string
	expectedDangerousSecretNames []string
}

func TestCheckForDangerousSecretNames(t *testing.T) {
	testCases := []dangerousSecretNameTestCase{
		{
			name: "Should not find any dangerous secret names",
			secrets: map[string]string{
				"MY_SECRET": "123",
			},
			expectedDangerousSecretNames: nil,
		},
		{
			name: "Should find a dangerous secret name",
			secrets: map[string]string{
				"DYLD_INSERT_LIBRARIES": "123",
				"MY_SECRET":             "123",
			},
			expectedDangerousSecretNames: []string{"DYLD_INSERT_LIBRARIES"},
		},
		{
			name: "Should find multiple dangerous secret names",
			secrets: map[string]string{
				"DYLD_INSERT_LIBRARIES": "123",
				"MY_SECRET":             "123",
				"LD_LIBRARY_PATH":       "123",
				"WINDIR":                "123",
				"PROMPT_COMMAND":        "123",
			},
			expectedDangerousSecretNames: []string{"DYLD_INSERT_LIBRARIES", "LD_LIBRARY_PATH", "WINDIR", "PROMPT_COMMAND"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := CheckForDangerousSecretNames(testCase.secrets)
			if testCase.expectedDangerousSecretNames == nil {
				assert.Nil(t, nil, err)
			} else {
				for _, v := range testCase.expectedDangerousSecretNames {
					assert.Contains(t, err.Error(), v)
				}
			}
		})
	}
}

type selectSecretsTestCase struct {
	name         string
	origMap      map[string]string
	keysToSelect []string
	expectedMap  map[string]string
	missingKeys  []string
}

func TestSelectSecrets(t *testing.T) {
	testCases := []selectSecretsTestCase{
		{
			name:         "Select one exisiting secret and two nonexistent secrets",
			origMap:      map[string]string{"MY_SECRET": "value"},
			keysToSelect: []string{"DEV", "LOGGING", "MY_SECRET"},
			expectedMap:  map[string]string{"MY_SECRET": "value"},
			missingKeys:  []string{"DEV", "LOGGING"},
		},
		{
			name:         "Select one secret",
			origMap:      map[string]string{"DEV": "true", "LOGGING": "true"},
			keysToSelect: []string{"DEV"},
			expectedMap:  map[string]string{"DEV": "true"},
		},
		{
			name:         "Select multiple secrets",
			origMap:      map[string]string{"DEV": "true", "LOGGING": "true", "MY_SECRET": "value", "PROD": "false"},
			keysToSelect: []string{"DEV", "LOGGING", "PROD"},
			expectedMap:  map[string]string{"DEV": "true", "LOGGING": "true", "PROD": "false"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filteredSecrets, err := SelectSecrets(testCase.origMap, testCase.keysToSelect)

			if testCase.missingKeys != nil {
				assert.NotNil(t, err)
				for _, missingKey := range testCase.missingKeys {
					assert.Contains(t, err.Error(), missingKey)
				}
			} else {
				assert.Nil(t, err)
			}

			assert.True(t, reflect.DeepEqual(filteredSecrets, testCase.expectedMap))
		})

	}
}

func TestSecretsToBytes(t *testing.T) {
	secrets := map[string]string{"S1": "foo", "SECRET2": "bar"}

	format := "env"
	bytes, err := SecretsToBytes(secrets, format, "")
	if !err.IsNil() || string(bytes) != strings.Join([]string{`S1="foo"`, `SECRET2="bar"`}, "\n") {
		t.Errorf("Unable to convert secrets to byte array in %s format", format)
	}

	format = "json"
	bytes, err = SecretsToBytes(secrets, format, "")
	if !err.IsNil() || string(bytes) != `{"S1":"foo","SECRET2":"bar"}` {
		t.Errorf("Unable to convert secrets to byte array in %s format", format)
	}

	format = "dotnet-json"
	bytes, err = SecretsToBytes(secrets, format, "")
	if !err.IsNil() || string(bytes) != `{"S1":"foo","Secret2":"bar"}` {
		t.Errorf("Unable to convert secrets to byte array in %s format", format)
	}
}
