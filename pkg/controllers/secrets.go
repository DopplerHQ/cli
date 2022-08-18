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
	"time"

	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
)

func GetSecretNames(config models.ScopedOptions) ([]string, Error) {
	utils.RequireValue("token", config.Token.Value)

	response, err := http.GetSecrets(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, nil, false, 0)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		return nil, Error{Err: parseErr, Message: "Unable to parse API response"}
	}

	var secretsNames []string
	for name := range secrets {
		secretsNames = append(secretsNames, name)
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
		mountData = []byte(RenderSecretsTemplate(templateBody, secrets))
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

func RenderSecretsTemplate(templateBody string, secretsMap map[string]string) string {
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
		utils.HandleError(err, "Unable to parse template text")
	}

	buffer := new(strings.Builder)
	err = template.Execute(buffer, secretsMap)
	if err != nil {
		utils.HandleError(err, "Unable to render template")
	}

	return buffer.String()
}
