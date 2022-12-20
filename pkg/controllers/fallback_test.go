/*
Copyright © 2022 Doppler <support@doppler.com>

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
	"testing"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/stretchr/testify/assert"
)

type fallbackFileHashTestCase struct {
	name          string
	secretNamesA  []string
	secretNamesB  []string
	expectedEqual bool
}

func TestGenerateFallbackFileHash(t *testing.T) {
	testCases := []fallbackFileHashTestCase{
		{
			name:          "unique hash per secret names",
			secretNamesA:  []string{"A"},
			secretNamesB:  []string{"B"},
			expectedEqual: false,
		},
		{
			name:          "sort secret names",
			secretNamesA:  []string{"A", "B"},
			secretNamesB:  []string{"B", "A"},
			expectedEqual: true,
		},
		{
			name:          "dedupe secret names",
			secretNamesA:  []string{"A", "B"},
			secretNamesB:  []string{"A", "B", "B"},
			expectedEqual: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			const token = "abc"
			const project = "abc"
			const config = "abc"

			if testCase.expectedEqual {
				assert.Equal(t, GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesA), GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesB))
			} else {
				assert.NotEqual(t, GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesA), GenerateFallbackFileHash(token, project, config, models.JSON, nil, testCase.secretNamesB))
			}
		})

	}

}
