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
package utils

import (
	"fmt"
	"sort"
	"strings"
)

func UpperCamel(name string) string {
	var parts []string
	for _, part := range strings.Split(name, "_") {
		if len(part) == 0 {
			continue
		}

		upperCamel := strings.ToUpper(part[0:1]) + strings.ToLower(part[1:])
		parts = append(parts, upperCamel)

	}
	return strings.Join(parts, "")
}

func DotNETNameTransform(name string) string {
	var parts []string
	for _, part := range strings.Split(name, "__") {
		parts = append(parts, UpperCamel(part))
	}
	return strings.Join(parts, ":")
}

func MapToEnvFormat(secrets map[string]string, wrapInQuotes bool) []string {
	var env []string
	for k, v := range secrets {
		if wrapInQuotes {
			v = strings.ReplaceAll(v, "\\", "\\\\")
			v = strings.ReplaceAll(v, "\"", "\\\"")
			env = append(env, fmt.Sprintf("%s=\"%s\"", k, v))
		} else {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// sort keys alphabetically for deterministic order
	sort.Slice(env, func(a, b int) bool {
		return env[a] < env[b]
	})

	return env
}

func MapToDotNETJSONFormat(secrets map[string]string) map[string]string {
	var dotnetJSON = make(map[string]string)
	for key, value := range secrets {
		keyTransform := DotNETNameTransform(key)
		dotnetJSON[keyTransform] = value
	}
	return dotnetJSON
}
