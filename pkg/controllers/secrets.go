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
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
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

type FallbackOptions struct {
	Enable             bool
	Path               string
	LegacyPath         string
	Readonly           bool
	Exclusive          bool
	ExitOnWriteFailure bool
	Passphrase         string
}

type MountOptions struct {
	Enable   bool
	Format   string
	Path     string
	Template string
	MaxReads int
}

func GetSecrets(config models.ScopedOptions) (map[string]models.ComputedSecret, Error) {
	utils.RequireValue("token", config.Token.Value)

	response, err := http.GetSecrets(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, nil, false, 0)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}
	secrets, parseErr := models.ParseSecrets(response)
	if parseErr != nil {
		return nil, Error{Err: parseErr, Message: "Unable to parse API response"}
	}

	return secrets, Error{}
}

func SetSecrets(config models.ScopedOptions, changeRequests []models.ChangeRequest) (map[string]models.ComputedSecret, Error) {
	utils.RequireValue("token", config.Token.Value)

	secrets, err := http.SetSecrets(config.APIHost.Value, utils.GetBool(config.VerifyTLS.Value, true), config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, nil, changeRequests)
	if !err.IsNil() {
		return nil, Error{Err: err.Unwrap(), Message: err.Message}
	}

	return secrets, Error{}
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

// SecretsToBytes converts secrets to byte array
func SecretsToBytes(secrets map[string]string, format string, templateBody string) ([]byte, Error) {
	if format == models.TemplateMountFormat {
		return []byte(RenderSecretsTemplate(templateBody, secrets)), Error{}
	}

	if format == models.EnvMountFormat {
		return []byte(strings.Join(utils.MapToEnvFormat(secrets, true), "\n")), Error{}
	}

	if format == models.JSONMountFormat {
		envStr, err := json.Marshal(secrets)
		if err != nil {
			return nil, Error{Err: err, Message: "Unable to marshal secrets to json"}
		}
		return envStr, Error{}
	}

	if format == models.DotNETJSONMountFormat {
		envStr, err := json.Marshal(utils.MapToDotNETJSONFormat(secrets))
		if err != nil {
			return nil, Error{Err: err, Message: "Unable to marshal .NET formatted secrets to json"}
		}
		return envStr, Error{}
	}

	return nil, Error{Err: fmt.Errorf("invalid mount format. Valid formats are %s", models.SecretsMountFormats)}
}

// MountSecrets mounts
func MountSecrets(secrets []byte, mountPath string, maxReads int) (string, func(), Error) {
	if !utils.SupportsNamedPipes {
		return "", nil, Error{Err: errors.New("This OS does not support mounting a secrets file")}
	}

	if mountPath == "" {
		return "", nil, Error{Err: errors.New("Mount path cannot be blank")}
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

	fifoCleanupStarted := false

	// cleanup named pipe on exit
	cleanupFIFO := func() {
		fifoCleanupStarted = true

		utils.LogDebug(fmt.Sprintf("Deleting secrets mount %s", mountPath))
		if err := os.Remove(mountPath); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// ignore
				return
			}

			utils.LogDebug("Unable to delete secrets mount")
			utils.LogError(err)
		}
	}

	utils.LogDebug(fmt.Sprintf("Mounting secrets to %s", mountPath))

	// open, write, and close the named pipe, repeatedly.
	// run as goroutine to prevent blocking later operations
	go func() {
		message := "Unable to mount secrets file"
		enableReadsLimit := maxReads > 0
		numReads := 0

		for {
			if enableReadsLimit && numReads >= maxReads {
				utils.LogDebug(fmt.Sprintf("Secrets mount has reached its limit of %d read(s)", maxReads))
				break
			}

			// this operation blocks until the FIFO is opened for reads
			f, err := os.OpenFile(mountPath, os.O_WRONLY, os.ModeNamedPipe) // #nosec G304
			if err != nil {
				// race: cleanup has already begun; no need to error
				if errors.Is(err, fs.ErrNotExist) && fifoCleanupStarted {
					break
				}
				cleanupFIFO()
				utils.HandleError(err, message)
			}

			numReads++
			utils.LogDebug("Secrets mount opened by reader")

			if _, err := f.Write(secrets); err != nil {
				// race: cleanup has already begun; no need to error
				if errors.Is(err, fs.ErrNotExist) && fifoCleanupStarted {
					break
				}
				cleanupFIFO()
				utils.HandleError(err, message)
			}

			if err := f.Close(); err != nil {
				// race: cleanup has already begun; no need to error
				if errors.Is(err, fs.ErrNotExist) && fifoCleanupStarted {
					break
				}
				cleanupFIFO()
				utils.HandleError(err, message)
			}

			// delay before re-opening file so reader can detect an EOF.
			// if we immediately re-open the file, the original reader will keep reading
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

func MissingSecrets(secrets map[string]string, secretsToInclude []string) []string {
	var missingSecrets []string
	for _, name := range secretsToInclude {
		if _, ok := secrets[name]; !ok {
			missingSecrets = append(missingSecrets, name)
		}
	}

	return missingSecrets
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

func ValidateSecrets(secrets map[string]string, secretsToInclude []string, exitOnMissingIncludedSecrets bool, mountOptions MountOptions) {
	if len(secretsToInclude) > 0 {
		missingSecrets := MissingSecrets(secrets, secretsToInclude)
		if len(missingSecrets) > 0 {
			err := fmt.Errorf("the following secrets you are trying to include do not exist in your config:\n- %v", strings.Join(missingSecrets, "\n- "))
			if exitOnMissingIncludedSecrets {
				utils.HandleError(err)
			} else {
				utils.LogWarning(err.Error())
			}
		}
	}

	// The potentially dangerous secret names only are only dangerous when they are set
	// as environment variables since they have the potential to change the default shell behavior.
	// When mounting the secrets into a file these are not dangerous
	if !mountOptions.Enable {
		if err := CheckForDangerousSecretNames(secrets); err != nil {
			utils.LogWarning(err.Error())
		}
	}
}

func PrepareSecrets(dopplerSecrets map[string]string, originalEnv []string, preserveEnv string, mountOptions MountOptions) ([]string, func()) {
	env := []string{}
	secrets := map[string]string{}
	var onExit func()
	if mountOptions.Enable {
		secrets = dopplerSecrets
		env = originalEnv

		secretsBytes, err := SecretsToBytes(secrets, mountOptions.Format, mountOptions.Template)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		absMountPath, handler, err := MountSecrets(secretsBytes, mountOptions.Path, mountOptions.MaxReads)
		if !err.IsNil() {
			utils.HandleError(err.Unwrap(), err.Message)
		}
		mountPath := absMountPath
		onExit = handler

		// export path to mounted file
		env = append(env, fmt.Sprintf("%s=%s", "DOPPLER_CLI_SECRETS_PATH", mountPath))
	} else {
		// remove any reserved keys from secrets
		reservedKeys := []string{"PATH", "PS1", "HOME"}
		for _, reservedKey := range reservedKeys {
			if _, found := dopplerSecrets[reservedKey]; found {
				utils.LogDebug(fmt.Sprintf("Ignoring reserved secret %s", reservedKey))
				delete(dopplerSecrets, reservedKey)
			}
		}

		existingEnvKeys := map[string]string{}
		for _, envVar := range originalEnv {
			// key=value format
			parts := strings.SplitN(envVar, "=", 2)
			key := parts[0]
			value := parts[1]
			existingEnvKeys[key] = value
		}

		if preserveEnv != "false" {
			secretsToPreserve := strings.Split(preserveEnv, ",")

			// use doppler secrets
			for name, value := range dopplerSecrets {
				secrets[name] = value
			}
			// then use existing env vars
			for name, value := range existingEnvKeys {
				if preserveEnv != "true" && !utils.Contains(secretsToPreserve, name) {
					continue
				}

				if _, found := secrets[name]; found {
					utils.LogDebug(fmt.Sprintf("Ignoring Doppler secret %s", name))
				}
				secrets[name] = value
			}
		} else {
			// use existing env vars
			for name, value := range existingEnvKeys {
				secrets[name] = value
			}
			// then use doppler secrets
			for name, value := range dopplerSecrets {
				secrets[name] = value
			}
		}

		for _, envVar := range utils.MapToEnvFormat(secrets, false) {
			env = append(env, envVar)
		}
	}

	return env, onExit
}

// fetchSecrets from Doppler and handle fallback file
func FetchSecrets(localConfig models.ScopedOptions, enableCache bool, fallbackOpts FallbackOptions, metadataPath string, nameTransformer *models.SecretsNameTransformer, dynamicSecretsTTL time.Duration, format models.SecretsFormat, secretNames []string) map[string]string {
	if fallbackOpts.Exclusive {
		if !fallbackOpts.Enable {
			utils.HandleError(errors.New("Conflict: unable to specify --no-fallback with --fallback-only"))
		}
		return readFallbackFile(fallbackOpts.Path, fallbackOpts.LegacyPath, fallbackOpts.Passphrase, false)
	}

	// this scenario likely isn't possible, but just to be safe, disable using cache when there's no metadata file
	enableCache = enableCache && metadataPath != ""
	etag := ""
	if enableCache {
		etag = getCacheFileETag(metadataPath, fallbackOpts.Path)
	}

	statusCode, respHeaders, response, httpErr := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, nameTransformer, etag, dynamicSecretsTTL, secretNames)
	if !httpErr.IsNil() {
		canUseFallback := statusCode != 401 && statusCode != 403 && statusCode != 404
		if !canUseFallback {
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, fallbackOpts.Path))
			os.Remove(fallbackOpts.Path)
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, fallbackOpts.LegacyPath))
			os.Remove(fallbackOpts.LegacyPath)
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, metadataPath))
			os.Remove(metadataPath)
		}

		if fallbackOpts.Enable && canUseFallback {
			utils.Log("Unable to fetch secrets from the Doppler API")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackOpts.Path, fallbackOpts.LegacyPath, fallbackOpts.Passphrase, false)
		}
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	if enableCache && statusCode == 304 {
		utils.LogDebug("Using cached secrets from fallback file")
		cache, err := SecretsCacheFile(fallbackOpts.Path, fallbackOpts.Passphrase)
		if !err.IsNil() {
			utils.LogDebugError(err.Unwrap())
			utils.LogDebug(err.Message)

			// we have to exit here as we don't have any secrets to parse
			utils.HandleError(err.Unwrap(), err.Message)
		}

		return cache
	}

	// ensure the response can be parsed before proceeding
	secrets, err := parseSecrets(response)
	if err != nil {
		utils.LogDebugError(err)

		if fallbackOpts.Enable {
			utils.Log("Unable to parse the Doppler API response")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackOpts.Path, fallbackOpts.LegacyPath, fallbackOpts.Passphrase, false)
		}
		utils.HandleError(err, "Unable to parse API response")
	}

	writeFallbackFile := fallbackOpts.Enable && !fallbackOpts.Readonly && nameTransformer == nil
	if writeFallbackFile {
		utils.LogDebug("Encrypting secrets")
		encryptedResponse, err := crypto.Encrypt(fallbackOpts.Passphrase, response, "base64")
		if err != nil {
			utils.HandleError(err, "Unable to encrypt your secrets. No fallback file has been written.")
		}

		utils.LogDebug(fmt.Sprintf("Writing to fallback file %s", fallbackOpts.Path))
		if err := utils.WriteFile(fallbackOpts.Path, []byte(encryptedResponse), utils.RestrictedFilePerms()); err != nil {
			utils.Log("Unable to write to fallback file")
			if fallbackOpts.ExitOnWriteFailure {
				utils.HandleError(err, "", strings.Join(WriteFailureMessage(), "\n"))
			} else {
				utils.LogDebugError(err)
			}
		}

		if enableCache {
			if etag := respHeaders.Get("etag"); etag != "" {
				hash := crypto.Hash(encryptedResponse)

				if err := WriteMetadataFile(metadataPath, etag, hash); !err.IsNil() {
					utils.LogDebugError(err.Unwrap())
					utils.LogDebug(err.Message)
				}
			} else {
				utils.LogDebug("API response does not contain ETag")
			}
		}
	}

	return secrets
}

func Run(cmd *cobra.Command, args []string, env []string, forwardSignals bool) (*exec.Cmd, error) {
	var c *exec.Cmd
	var err error

	if cmd.Flags().Changed("command") {
		command := cmd.Flag("command").Value.String()
		c, err = utils.RunCommandString(command, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals)
	} else {
		c, err = utils.RunCommand(args, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals)
	}

	return c, err
}

func readFallbackFile(path string, legacyPath string, passphrase string, silent bool) map[string]string {
	// avoid re-logging if re-running for legacy file
	// TODO remove this when removing legacy path support
	if !silent {
		utils.Log("Reading secrets from fallback file")
	}
	utils.LogDebug(fmt.Sprintf("Using fallback file %s", path))

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// attempt to read from the legacy path, in case the fallback file was created with an older version of the CLI
			// TODO remove this when releasing CLI v4 (DPLR-435)
			if legacyPath != "" {
				return readFallbackFile(legacyPath, "", passphrase, true)
			}

			utils.HandleError(errors.New("The fallback file does not exist"))
		}

		utils.HandleError(err, "Unable to read fallback file")
	}

	response, err := ioutil.ReadFile(path) // #nosec G304
	if err != nil {
		utils.HandleError(err, "Unable to read fallback file")
	}

	utils.LogDebug("Decrypting fallback file")
	decryptedSecrets, err := crypto.Decrypt(passphrase, response)
	if err != nil {
		var msg []string
		msg = append(msg, "")
		msg = append(msg, "=== More Info ===")
		msg = append(msg, "")
		msg = append(msg, color.Green.Render("Why did decryption fail?"))
		msg = append(msg, "The most common cause of decryption failure is using an incorrect passphrase.")
		msg = append(msg, "The default passphrase is computed using your token, project, and config.")
		msg = append(msg, "You must use the same token, project, and config that you used when saving the backup file.")
		msg = append(msg, "")
		msg = append(msg, color.Green.Render("What should I do now?"))
		msg = append(msg, "Ensure you are using the same scope that you used when creating the fallback file.")
		msg = append(msg, "Alternatively, manually specify your configuration using the appropriate flags (e.g. --project).")
		msg = append(msg, "")
		msg = append(msg, "Run 'doppler run --help' for more info.")
		msg = append(msg, "")

		utils.HandleError(err, "Unable to decrypt fallback file", strings.Join(msg, "\n"))
	}

	secrets, err := parseSecrets([]byte(decryptedSecrets))
	if err != nil {
		utils.HandleError(err, "Unable to parse fallback file")
	}

	return secrets
}

func WriteFailureMessage() []string {
	var msg []string

	msg = append(msg, "")
	msg = append(msg, "=== More Info ===")
	msg = append(msg, "")
	msg = append(msg, color.Green.Render("Why did doppler exit?"))
	msg = append(msg, "Doppler failed to make a local backup of your secrets, known as a fallback file.")
	msg = append(msg, "The most common cause for this is insufficient permissions, including trying to use a fallback file created by a different user.")
	msg = append(msg, "")
	msg = append(msg, color.Green.Render("Why does this matter?"))
	msg = append(msg, "Without the fallback file, your secrets would be inaccessible in the event of a network outage or Doppler downtime.")
	msg = append(msg, "This could mean your development is blocked, or it could mean that your production services can't start up.")
	msg = append(msg, "")
	msg = append(msg, color.Green.Render("What should I do now?"))
	msg = append(msg, "1. You can change the location of the fallback file using the '--fallback' flag.")
	msg = append(msg, "2. You can attempt to debug and fix the local error causing the write failure.")
	msg = append(msg, "3. You can choose to ignore this error using the '--no-exit-on-write-failure' flag, but be forewarned that this is probably a really bad idea.")
	msg = append(msg, "")
	msg = append(msg, "Run 'doppler run --help' for more info.")
	msg = append(msg, "")

	return msg
}

func parseSecrets(response []byte) (map[string]string, error) {
	secrets := map[string]string{}
	err := json.Unmarshal(response, &secrets)
	return secrets, err
}

func getCacheFileETag(metadataPath string, cachePath string) string {
	metadata, Err := MetadataFile(metadataPath)
	if !Err.IsNil() {
		utils.LogDebugError(Err.Unwrap())
		utils.LogDebug(Err.Message)
		return ""
	}

	if metadata.Hash == "" {
		return metadata.ETag
	}

	// verify hash
	cacheFileBytes, err := ioutil.ReadFile(cachePath) // #nosec G304
	if err == nil {
		cacheFileContents := string(cacheFileBytes)
		hash := crypto.Hash(cacheFileContents)

		if hash == metadata.Hash {
			return metadata.ETag
		}

		utils.LogDebug("Fallback file failed hash check, ignoring cached secrets")
	}

	return ""
}
