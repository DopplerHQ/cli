/*
Copyright © 2026 Doppler <support@doppler.com>

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

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestPreserveEnvFlagParsing(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
		posArgs  []string
	}{
		{"default", []string{}, "false", []string{}},
		{"bare flag means true", []string{"--preserve-env"}, "true", []string{}},
		{"equals form sets list", []string{"--preserve-env=FOO,BAR"}, "FOO,BAR", []string{}},
		{"space form doesn't consume the value", []string{"--preserve-env", "FOO"}, "true", []string{"FOO"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// rebuild the flag on a fresh set so parsing doesn't mutate runCmd's global state
			original := runCmd.Flags().Lookup("preserve-env")
			flags := (&cobra.Command{}).Flags()
			flags.String(original.Name, original.DefValue, original.Usage)
			flags.Lookup(original.Name).NoOptDefVal = original.NoOptDefVal
			assert.NoError(t, flags.Parse(tc.args))
			assert.Equal(t, tc.expected, flags.Lookup("preserve-env").Value.String())
			assert.Equal(t, tc.posArgs, flags.Args())
		})
	}
}
