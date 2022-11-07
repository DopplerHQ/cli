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

	"github.com/stretchr/testify/assert"
)

type filterOutSecretTestCase struct {
	name         string
	origMap      map[string]string
	keysToSelect []string
	expectedMap  map[string]string
}

func TestFilterMap(t *testing.T) {
	testCases := []filterOutSecretTestCase{
		{
			name:         "Select no elements",
			origMap:      map[string]string{"title": "a title"},
			keysToSelect: []string{""},
			expectedMap:  map[string]string{},
		},
		{
			name:         "Select nonexistent key",
			origMap:      map[string]string{"title": "a title"},
			keysToSelect: []string{"key"},
			expectedMap:  map[string]string{},
		},
		{
			name:         "Select one element",
			origMap:      map[string]string{"title": "a title", "address": "an address"},
			keysToSelect: []string{"title"},
			expectedMap:  map[string]string{"title": "a title"},
		},
		{
			name:         "Select one element and a nonexistent key",
			origMap:      map[string]string{"title": "a title", "address": "an address"},
			keysToSelect: []string{"title", "notPresent"},
			expectedMap:  map[string]string{"title": "a title"},
		},
		{
			name:         "Select multiple elements",
			origMap:      map[string]string{"title": "a title", "address": "an address"},
			keysToSelect: []string{"title", "address"},
			expectedMap:  map[string]string{"title": "a title", "address": "an address"},
		},
		{
			name:         "Select more elements",
			origMap:      map[string]string{"title": "a title", "address": "an address", "phoneNumber": "a phone number", "city": "a city"},
			keysToSelect: []string{"title", "address", "city", "boom"},
			expectedMap:  map[string]string{"title": "a title", "address": "an address", "city": "a city"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			filteredSecrets := FilterMap(testCase.origMap, testCase.keysToSelect)
			assert.True(t, reflect.DeepEqual(filteredSecrets, testCase.expectedMap))
		})

	}

}

func BenchmarkFilterMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FilterMap(map[string]string{"title": "a title", "address": "an address", "phoneNumber": "a phone number", "city": "a city"},
			[]string{"title", "address", "phoneNumber", "city"},
		)
	}
}
