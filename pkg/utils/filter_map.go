/*
Copyright © 2020 Doppler <support@doppler.com>

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

// FilterMap returns a new map[string]T containing origMap keys that match aginst the keysToSelect
// if a key in keysToSelect is not present in origMap it will be silently ignored
func FilterMap[T any](origMap map[string]T, keysToSelect []string) map[string]T {
	filteredMap := map[string]T{}

	for _, keyToSelect := range keysToSelect {
		if _, ok := origMap[keyToSelect]; ok {
			filteredMap[keyToSelect] = origMap[keyToSelect]
		}
	}

	return filteredMap
}
