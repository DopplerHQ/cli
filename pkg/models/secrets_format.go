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
package models

// SecretsFormat the format secrets should use
type SecretsFormat int

// the source of the value
const (
	JSON SecretsFormat = iota
	DOTNET_JSON
	ENV
	YAML
	DOCKER
	ENV_NO_QUOTES
)

var SecretFormats = []string{"json", "dotnet-json", "env", "yaml", "docker", "env-no-quotes"}

func (s SecretsFormat) String() string {
	return SecretFormats[s]
}

// OutputFile the default secrets file name
func (s SecretsFormat) OutputFile() string {
	return [...]string{"doppler.json", "appsettings.json", "doppler.env", "secrets.yaml", "doppler.env", "doppler.env"}[s]
}

// SecretsFormatList list of supported secrets formats
var SecretsFormatList []SecretsFormat

func init() {
	SecretsFormatList = append(SecretsFormatList, JSON)
	SecretsFormatList = append(SecretsFormatList, DOTNET_JSON)
	SecretsFormatList = append(SecretsFormatList, ENV)
	SecretsFormatList = append(SecretsFormatList, YAML)
	SecretsFormatList = append(SecretsFormatList, DOCKER)
	SecretsFormatList = append(SecretsFormatList, ENV_NO_QUOTES)
}

// TemplateMountFormat is a special mount format that uses JSON from the backend
// piped through a user-defined local template file. It is not a "real" backend format.
const TemplateMountFormat = "template"

// SecretsMountFormats lists valid formats for mounting secrets.
// This includes backend formats (excluding yaml) plus the special template format.
var SecretsMountFormats = []string{
	ENV.String(),
	JSON.String(),
	DOTNET_JSON.String(),
	TemplateMountFormat,
	ENV_NO_QUOTES.String(),
	DOCKER.String(),
}

// IsValidMountFormat checks if a format string is valid for mounting
func IsValidMountFormat(format string) bool {
	for _, f := range SecretsMountFormats {
		if f == format {
			return true
		}
	}
	return false
}

// GetMountSecretsFormat returns the SecretsFormat for a mount format string.
// Returns JSON for template format (since template uses JSON from the backend).
// Returns false if the format is not valid.
func GetMountSecretsFormat(format string) (SecretsFormat, bool) {
	if format == TemplateMountFormat {
		return JSON, true
	}
	for _, f := range SecretsFormatList {
		if f.String() == format {
			return f, true
		}
	}
	return JSON, false
}
