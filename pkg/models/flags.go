/*
Copyright © 2023 Doppler <support@doppler.com>

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
package models

const (
	FlagAnalytics   string = "analytics"
	FlagEnvWarning  string = "env-warning"
	FlagUpdateCheck string = "update-check"
)

type Flags struct {
	Analytics   *bool `yaml:"analytics,omitempty"`
	EnvWarning  *bool `yaml:"env-warning,omitempty"`
	UpdateCheck *bool `yaml:"update-check,omitempty"`
}

var flags = []string{
	FlagAnalytics,
	FlagEnvWarning,
	FlagUpdateCheck,
}

func GetFlags() []string {
	return flags
}
