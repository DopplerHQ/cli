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

import "strings"

// ParseEnvStrings returns a new map[string]string created by parsing env strings (in `key=value` format).
func ParseEnvStrings(envStrings []string) map[string]string {
	env := map[string]string{}
	for _, envVar := range envStrings {
		// key=value format. a parent process can pass entries without "=", which aren't valid env vars
		key, value, found := strings.Cut(envVar, "=")
		if !found {
			continue
		}
		env[key] = value
	}

	return env
}
