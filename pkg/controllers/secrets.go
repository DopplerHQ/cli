/*
Copyright © 2021 Doppler <support@doppler.com>

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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"text/template/parse"
	"time"

	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

// Documentation about potentially dangerous secret names can be found here: https://docs.doppler.com/docs/accessing-secrets#injection
var dangerousSecretNames = [...]string{
	// Operating Systems environment variable names
	// Linux
	"PROMPT_COMMAND",
	"LD_PRELOAD",
	"LD_LIBRARY_PATH",
	// Windows
	"WINDIR",
	"USERPROFILE",
	// MacOS
	"DYLD_INSERT_LIBRARIES",

	// Language Interpreters environment variable names
	// Perl & Python
	"PERL5OPT",
	// Python
	"PYTHONWARNINGS",
	"BROWSER",
	// PHP
	"HOSTNAME",
	"PHPRC",
	// NodeJS
	"NODE_VERSION",
	"NODE_OPTIONS",
}

func GetSecretNames(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	secretsNames, err := http.GetSecretNames(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, false)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	sort.Strings(secretsNames)

	return secretsNames, Error{}
}

func MountSecrets(secrets map[string]string, format string, mountPath string, maxReads int, templateBody string) (string, func(), Error) {
	if !utils.SupportsNamedPipes {
		return "", nil, Error{Err: errors.New("This OS does not support mounting a secrets file")}
	}

	if mountPath == "" {
		return "", nil, Error{Err: errors.New("Mount path cannot be blank")}
	}

	var mountData []byte
	if format == models.TemplateMountFormat {
		template, err := ParseSecretsTemplate(templateBody)
		if !err.IsNil() {
			return "", nil, err
		}
		mountData = []byte(RenderSecretsTemplate(template, secrets))
	} else if format == models.EnvMountFormat {
		mountData = []byte(strings.Join(utils.MapToEnvFormat(secrets, true), "\n"))
	} else if format == models.JSONMountFormat {
		envStr, err := json.Marshal(secrets)
		if err != nil {
			return "", nil, Error{Err: err, Message: "Unable to marshall secrets to json"}
		}
		mountData = envStr
	} else if format == models.DotNETJSONMountFormat {
		envStr, err := json.Marshal(utils.MapToDotNETJSONFormat(secrets))
		if err != nil {
			return "", nil, Error{Err: err, Message: "Unable to marshall .NET formatted secrets to json"}
		}
		mountData = envStr
	} else {
		return "", nil, Error{Err: fmt.Errorf("invalid mount format. Valid formats are %s", models.SecretsMountFormats)}

	}

	// convert mount path to absolute path
	var err error
	mountPath, err = filepath.Abs(mountPath)
	if err != nil {
		return "", nil, Error{Err: err, Message: "Unable to resolve mount path"}
	}

	if utils.Exists(mountPath) {
		return "", nil, Error{Err: errors.New("The mount path already exists")}
	}

	if err := utils.CreateNamedPipe(mountPath, 0600); err != nil {
		return "", nil, Error{Err: err, Message: "Unable to mount secrets file"}
	}

	// cleanup named pipe on exit
	cleanupFIFO := func() {
		if utils.Exists(mountPath) {
			utils.LogDebug(fmt.Sprintf("Deleting secrets mount %s", mountPath))
			if err := os.Remove(mountPath); err != nil {
				utils.LogDebug("Unable to delete secrets mount")
				utils.LogError(err)
			}
		}
	}

	// prevent this from blocking later operations
	go func() {
		message := "Unable to mount secrets file"
		enableReadsLimit := maxReads > 0
		numReads := 0

		utils.LogDebug(fmt.Sprintf("Mounting secrets in %s format to %s", format, mountPath))

		for {
			if enableReadsLimit && numReads >= maxReads {
				utils.LogDebug(fmt.Sprintf("Secrets mount has reached its limit of %d read(s)", maxReads))
				break
			}

			numReads++

			// this operation blocks until the FIFO is opened for reads
			f, err := os.OpenFile(mountPath, os.O_WRONLY, os.ModeNamedPipe) // #nosec G304
			if err != nil {
				utils.HandleError(err, message)
			}

			if _, err := f.Write(mountData); err != nil {
				utils.HandleError(err, message)
			}

			if err := f.Close(); err != nil {
				utils.HandleError(err, message)
			}

			// delay before re-opening file so reader can detect an EOF
			time.Sleep(time.Millisecond * 10)
		}

		cleanupFIFO()
	}()

	return mountPath, cleanupFIFO, Error{}
}

func ReadTemplateFile(filePath string) string {
	templateFilePath, err := utils.GetFilePath(filePath)
	if err != nil {
		utils.HandleError(err, "Unable to parse template file path")
	}

	var templateFile []byte
	templateFile, err = ioutil.ReadFile(templateFilePath) // #nosec G304
	if err != nil {
		utils.HandleError(err, "Unable to read template file")
	}
	return string(templateFile)
}

func TemplateFields(t *template.Template) []string {
	return nodeFields(t.Tree.Root, nil)
}

func nodeFields(node parse.Node, res []string) []string {
	if node.Type() == parse.NodeAction {
		res = append(res, node.String())
	}

	if ln, ok := node.(*parse.ListNode); ok {
		for _, n := range ln.Nodes {
			res = nodeFields(n, res)
		}
	}
	return res
}

func ParseSecretsTemplate(templateBody string) (*template.Template, Error) {
	funcs := map[string]interface{}{
		"tojson": func(value interface{}) (string, error) {
			body, err := json.Marshal(value)
			if err != nil {
				return "", err
			}
			return string(body), nil
		},
		"fromjson": func(value string) (interface{}, error) {
			var result interface{}
			err := json.Unmarshal([]byte(value), &result)
			if err != nil {
				return "", err
			}
			return result, nil
		},
	}
	template, err := template.New("Secrets").Funcs(funcs).Parse(templateBody)
	if err != nil {
		return nil, Error{Err: err, Message: "Unable to parse template text"}
	}

	return template, Error{}
}

func RenderSecretsTemplate(template *template.Template, secretsMap map[string]string) string {
	buffer := new(strings.Builder)
	if err := template.Execute(buffer, secretsMap); err != nil {
		utils.HandleError(err, "Unable to render template")
	}

	return buffer.String()
}

func SelectSecrets(secrets map[string]string, secretsToSelect []string) (map[string]string, error) {
	selectedSecrets := utils.FilterMap(secrets, secretsToSelect)
	nonExistentSecretNames := []string{}

	for _, secretName := range secretsToSelect {
		if _, found := selectedSecrets[secretName]; !found {
			nonExistentSecretNames = append(nonExistentSecretNames, secretName)
		}
	}

	var err error
	if len(nonExistentSecretNames) > 0 {
		err = fmt.Errorf("the following secrets you are trying to include do not exist in your config:\n- %v", strings.Join(nonExistentSecretNames, "\n- "))
	}

	return selectedSecrets, err
}

// CheckForDangerousSecretNames checks for potential dangerous secret names.
// Documentation about potentially dangerous secret names can be found here: https://docs.doppler.com/docs/accessing-secrets#injection
func CheckForDangerousSecretNames(secrets map[string]string) error {
	dangerousSecretNamesFound := []string{}

	for _, dangerousName := range dangerousSecretNames {
		if _, ok := secrets[dangerousName]; ok {
			dangerousSecretNamesFound = append(dangerousSecretNamesFound, dangerousName)
		}
	}

	if len(dangerousSecretNamesFound) > 0 {
		return fmt.Errorf("your config contains the following potentially dangerous secret names (https://docs.doppler.com/docs/accessing-secrets#injection):\n- %s", strings.Join(dangerousSecretNamesFound, "\n- "))
	}

	return nil
}
