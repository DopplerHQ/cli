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
	"reflect"
	"testing"
)

type testCase struct {
	name          string
	nameTransform string
}

func TestUpperCamel(t *testing.T) {
	testCases := []testCase{
		{"TEST", "Test"},
		{"TEST_", "Test"},
		{"TEST_SECRET", "TestSecret"},
		{"TEST_SECRET_NAME", "TestSecretName"},
		{"TEST__SECRET", "TestSecret"},
	}

	for _, testCase := range testCases {
		nameTransform := UpperCamel(testCase.name)
		if testCase.nameTransform != nameTransform {
			t.Errorf("Expected '%s' to be '%s' but got '%s'", testCase.name, testCase.nameTransform, nameTransform)
		}
	}
}

func TestDotNETNameTransform(t *testing.T) {
	testCases := []testCase{
		{"TEST", "Test"},
		{"TEST_", "Test"},
		{"TEST_SECRET", "TestSecret"},
		{"TEST_SECRET_NAME", "TestSecretName"},
		{"TEST__SECRET", "Test:Secret"},
	}

	for _, testCase := range testCases {
		nameTransform := DotNETNameTransform(testCase.name)
		if testCase.nameTransform != nameTransform {
			t.Errorf("Expected '%s' to be '%s' but got '%s'", testCase.name, testCase.nameTransform, nameTransform)
		}
	}
}

func TestMapToDotNETJSONFormat(t *testing.T) {
	secrets := map[string]string{
		"TEST":             "",
		"TEST_SECRET":      "",
		"TEST_SECRET_NAME": "",
		"TEST__SECRET":     "",
	}

	transformedSecrets := map[string]string{
		"Test":           "",
		"TestSecret":     "",
		"TestSecretName": "",
		"Test:Secret":    "",
	}

	transformedSecretsResult := MapToDotNETJSONFormat(secrets)

	if !reflect.DeepEqual(transformedSecrets, transformedSecretsResult) {
		t.Errorf("Expected '%s' to be '%s' but got '%s'", secrets, transformedSecrets, transformedSecretsResult)
	}
}
