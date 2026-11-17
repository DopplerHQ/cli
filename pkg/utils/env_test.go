/*
Copyright © 2024 Doppler <support@doppler.com>

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

	"github.com/stretchr/testify/assert"
)

func TestParseEnvStrings(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  map[string]string
	}{
		{name: "nil input", want: map[string]string{}},
		{name: "empty input", input: []string{}, want: map[string]string{}},
		{name: "valid entries", input: []string{"FIRST=one", "SECOND=two"}, want: map[string]string{"FIRST": "one", "SECOND": "two"}},
		{name: "missing separator", input: []string{"NOEQ"}, want: map[string]string{}},
		{name: "empty entry", input: []string{""}, want: map[string]string{}},
		{name: "mixed entries", input: []string{"BEFORE=one", "NOEQ", "", "AFTER=two"}, want: map[string]string{"BEFORE": "one", "AFTER": "two"}},
		{name: "empty value", input: []string{"EMPTY="}, want: map[string]string{"EMPTY": ""}},
		{name: "value with equals", input: []string{"VALUE=one=two="}, want: map[string]string{"VALUE": "one=two="}},
		{name: "last duplicate wins", input: []string{"KEY=first", "KEY=last"}, want: map[string]string{"KEY": "last"}},
		{name: "empty key retains existing behavior", input: []string{"=value"}, want: map[string]string{"": "value"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got map[string]string
			if assert.NotPanics(t, func() { got = ParseEnvStrings(tt.input) }) {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
