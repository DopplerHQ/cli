/*
Copyright © 2019 Doppler <support@doppler.com>

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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var defaultFallbackDir string

const defaultFallbackFileMaxAge = 14 * 24 * time.Hour // 14 days

type fallbackOptions struct {
	enable             bool
	path               string
	legacyPath         string
	readonly           bool
	exclusive          bool
	exitOnWriteFailure bool
	passphrase         string
}

var secretsToInclude []string

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command with secrets injected into the environment",
	Long: `Run a command with secrets injected into the environment.
Secrets can also be mounted to an ephemeral file using the --mount flag.

To view the CLI's active configuration, run ` + "`doppler configure debug`",
	Example: `doppler run -- YOUR_COMMAND --YOUR-FLAG
doppler run --command "YOUR_COMMAND && YOUR_OTHER_COMMAND"
doppler run --mount secrets.json -- cat secrets.json`,
	Args: func(cmd *cobra.Command, args []string) error {
		// The --command flag and args are mututally exclusive
		usingCommandFlag := cmd.Flags().Changed("command")
		if usingCommandFlag {
			command := cmd.Flag("command").Value.String()
			if command == "" {
				return errors.New("--command flag requires a value")
			}

			if len(args) > 0 {
				return errors.New("arg(s) may not be set when using --command flag")
			}
		} else if len(args) == 0 {
			return errors.New("requires at least 1 arg(s), received 0")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		enableFallback := !utils.GetBoolFlag(cmd, "no-fallback")
		enableCache := enableFallback && !utils.GetBoolFlag(cmd, "no-cache")
		fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
		fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
		exitOnWriteFailure := !utils.GetBoolFlag(cmd, "no-exit-on-write-failure")
		preserveEnv := cmd.Flag("preserve-env").Value.String()
		forwardSignals := utils.GetBoolFlag(cmd, "forward-signals")
		localConfig := configuration.LocalConfig(cmd)
		dynamicSecretsTTL := utils.GetDurationFlag(cmd, "dynamic-ttl")

		utils.RequireValue("token", localConfig.Token.Value)

		if cmd.Flags().Changed("only-secrets") && len(secretsToInclude) == 0 {
			utils.HandleError(fmt.Errorf("you must specify secrets when using --only-secrets"))
		}

		nameTransformerString := cmd.Flag("name-transformer").Value.String()
		var nameTransformer *models.SecretsNameTransformer
		if nameTransformerString != "" {
			nameTransformer = models.SecretsNameTransformerMap[nameTransformerString]
			if nameTransformer == nil || !nameTransformer.EnvCompat {
				utils.HandleError(fmt.Errorf("invalid name transformer. Valid transformers are %s", validEnvCompatNameTransformersList))
			}
		}

		const format = models.JSON
		fallbackPath := ""
		legacyFallbackPath := ""
		metadataPath := ""
		if enableFallback {
			fallbackPath, legacyFallbackPath = initFallbackDir(cmd, localConfig, format, nameTransformer, secretsToInclude, exitOnWriteFailure)
		}
		if enableCache {
			metadataPath = controllers.MetadataFilePath(localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, nameTransformer, secretsToInclude)
		}

		passphrase := getPassphrase(cmd, "passphrase", localConfig)
		if passphrase == "" {
			utils.HandleError(errors.New("invalid passphrase"))
		}

		if !enableFallback {
			flags := []string{"fallback", "fallback-only", "fallback-readonly", "no-exit-on-write-failure", "passphrase"}
			for _, flag := range flags {
				if cmd.Flags().Changed(flag) {
					utils.LogWarning(fmt.Sprintf("--%s has no effect when the fallback file is disabled", flag))
				}
			}
		}

		fallbackOpts := fallbackOptions{
			enable:             enableFallback,
			path:               fallbackPath,
			legacyPath:         legacyFallbackPath,
			readonly:           fallbackReadonly,
			exclusive:          fallbackOnly,
			exitOnWriteFailure: exitOnWriteFailure,
			passphrase:         passphrase,
		}

		mountPath := cmd.Flag("mount").Value.String()
		mountFormatString := cmd.Flag("mount-format").Value.String()
		mountTemplate := cmd.Flag("mount-template").Value.String()
		maxReads := utils.GetIntFlag(cmd, "mount-max-reads", 32)
		// only auto-detect the format if it hasn't been explicitly specified
		shouldAutoDetectFormat := !cmd.Flags().Changed("mount-format")
		shouldMountFile := mountPath != ""
		shouldMountTemplate := mountTemplate != ""

		var mountFormat string
		if mountFormatVal, ok := models.SecretsMountFormatMap[mountFormatString]; ok {
			mountFormat = mountFormatVal
		} else {
			utils.HandleError(fmt.Errorf("Invalid mount format. Valid formats are %s", models.SecretsMountFormats))
		}

		if preserveEnv != "false" {
			if shouldMountFile {
				utils.LogWarning("--preserve-env has no effect when used with --mount")
			} else {
				utils.LogWarning("Ignoring Doppler secrets already defined in the environment due to --preserve-env flag")
			}
		}

		if shouldMountTemplate && !shouldMountFile {
			utils.HandleError(errors.New("--mount-template must be used with --mount"))
		}

		originalEnv := os.Environ()
		existingEnvKeys := map[string]string{}
		for _, envVar := range originalEnv {
			// key=value format
			parts := strings.SplitN(envVar, "=", 2)
			key := parts[0]
			value := parts[1]
			existingEnvKeys[key] = value
		}

		var templateBody string
		if shouldMountFile {
			if shouldAutoDetectFormat {
				if shouldMountTemplate {
					mountFormat = models.TemplateMountFormat
					utils.LogDebug(fmt.Sprintf("Detected %s format", mountFormat))
				} else if utils.IsDotNETSettingsFile(mountPath) {
					mountFormat = models.DotNETJSONMountFormat
					utils.LogDebug(fmt.Sprintf("Detected %s format", mountFormat))
				} else if strings.HasSuffix(mountPath, ".env") {
					mountFormat = models.EnvMountFormat
					utils.LogDebug(fmt.Sprintf("Detected %s format", mountFormat))
				} else if strings.HasSuffix(mountPath, ".json") {
					mountFormat = models.JSONMountFormat
					utils.LogDebug(fmt.Sprintf("Detected %s format", mountFormat))
				} else {
					parts := strings.Split(mountPath, ".")
					detectedFormat := parts[len(parts)-1]
					utils.LogWarning(fmt.Sprintf("Detected \"%s\" file format, which is not supported. Using default JSON format for mounted secrets", detectedFormat))
				}
			}

			utils.LogDebug(fmt.Sprintf("Using %s format", mountFormat))

			if shouldMountTemplate {
				if mountFormat != models.TemplateMountFormat {
					utils.HandleError(errors.New("--mount-template can only be used with --mount-format=template"))
				}
				templateBody = controllers.ReadTemplateFile(mountTemplate)
			} else if mountFormat == models.TemplateMountFormat {
				utils.HandleError(errors.New("--mount-template must be specified when using --mount-format=template"))
			}
		}

		// retrieve secrets
		dopplerSecrets := fetchSecrets(localConfig, enableCache, fallbackOpts, metadataPath, nameTransformer, dynamicSecretsTTL, format, secretsToInclude)

		// The potentially dangerous secret names only are only dangerous when they are set
		// as environment variables since they have the potential to change the default shell behavior.
		// When mounting the secrets into a file these are not dangerous
		if !shouldMountFile {
			if err := controllers.CheckForDangerousSecretNames(dopplerSecrets); err != nil {
				utils.LogWarning(err.Error())
			}
		}

		if len(secretsToInclude) > 0 {
			noExitOnMissingIncludedSecrets := cmd.Flags().Changed("no-exit-on-missing-only-secrets")
			missingSecrets := controllers.MissingSecrets(dopplerSecrets, secretsToInclude)
			if len(missingSecrets) > 0 {
				err := fmt.Errorf("the following secrets you are trying to include do not exist in your config:\n- %v", strings.Join(missingSecrets, "\n- "))
				if noExitOnMissingIncludedSecrets {
					utils.LogWarning(err.Error())
				} else {
					utils.HandleError(err)
				}
			}
		}

		env := []string{}
		secrets := map[string]string{}
		var onExit func()
		if shouldMountFile {
			secrets = dopplerSecrets
			env = originalEnv

			secretsBytes, err := controllers.SecretsToBytes(secrets, mountFormat, templateBody)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}
			absMountPath, handler, err := controllers.MountSecrets(secretsBytes, mountPath, maxReads)
			if !err.IsNil() {
				utils.HandleError(err.Unwrap(), err.Message)
			}
			mountPath = absMountPath
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

					if _, found := secrets[name]; found == true {
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

		exitCode := 0
		var err error

		if cmd.Flags().Changed("command") {
			command := cmd.Flag("command").Value.String()
			exitCode, err = utils.RunCommandString(command, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals, onExit)
		} else {
			exitCode, err = utils.RunCommand(args, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals, onExit)
		}

		if err != nil {
			if strings.HasPrefix(err.Error(), "exec") || strings.HasPrefix(err.Error(), "fork/exec") {
				utils.LogError(err)
			}
			utils.LogDebugError(err)
		}

		os.Exit(exitCode)
	},
}

var runCleanCmd = &cobra.Command{
	Use:     "clean",
	Short:   "Delete old fallback files",
	Long:    `Delete fallback files older than the max age from the default directory`,
	Example: `doppler run clean --max-age=24h`,
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		maxAge := utils.GetDurationFlag(cmd, "max-age")
		dryRun := utils.GetBoolFlag(cmd, "dry-run")
		all := utils.GetBoolFlag(cmd, "all")

		utils.LogDebug(fmt.Sprintf("Using fallback directory %s", defaultFallbackDir))

		if _, err := os.Stat(defaultFallbackDir); err != nil {
			if os.IsNotExist(err) {
				utils.LogDebug("Fallback directory does not exist")
				utils.Print("Nothing to clean")
				return
			}

			utils.HandleError(err, "Unable to read fallback directory")
		}

		entries, err := ioutil.ReadDir(defaultFallbackDir)
		if err != nil {
			utils.HandleError(err, "Unable to read fallback directory")
		}

		deleted := 0
		now := time.Now()

		action := "Deleted"
		if dryRun {
			action = "Would have deleted"
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			delete := false
			if all {
				delete = true
			} else {
				validUntil := entry.ModTime().Add(maxAge)
				if validUntil.Before(now) {
					delete = true
				}
			}

			if delete {
				file := filepath.Join(defaultFallbackDir, entry.Name())
				utils.LogDebug(fmt.Sprintf("%s %s", action, file))

				if dryRun {
					deleted++
					continue
				}

				err := os.Remove(file)
				if err != nil {
					// don't exit
					utils.Log(fmt.Sprintf("Unable to delete fallback file %s\n", file))
					utils.LogDebugError(err)
				} else {
					deleted++
				}
			}
		}

		if deleted == 1 {
			utils.Print(fmt.Sprintf("%s %d fallback file\n", action, deleted))
		} else {
			utils.Print(fmt.Sprintf("%s %d fallback files\n", action, deleted))
		}
	},
}

// fetchSecrets from Doppler and handle fallback file
func fetchSecrets(localConfig models.ScopedOptions, enableCache bool, fallbackOpts fallbackOptions, metadataPath string, nameTransformer *models.SecretsNameTransformer, dynamicSecretsTTL time.Duration, format models.SecretsFormat, secretNames []string) map[string]string {
	if fallbackOpts.exclusive {
		if !fallbackOpts.enable {
			utils.HandleError(errors.New("Conflict: unable to specify --no-fallback with --fallback-only"))
		}
		return readFallbackFile(fallbackOpts.path, fallbackOpts.legacyPath, fallbackOpts.passphrase, false)
	}

	// this scenario likely isn't possible, but just to be safe, disable using cache when there's no metadata file
	enableCache = enableCache && metadataPath != ""
	etag := ""
	if enableCache {
		etag = getCacheFileETag(metadataPath, fallbackOpts.path)
	}

	statusCode, respHeaders, response, httpErr := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, format, nameTransformer, etag, dynamicSecretsTTL, secretNames)
	if !httpErr.IsNil() {
		canUseFallback := statusCode != 401 && statusCode != 403 && statusCode != 404
		if !canUseFallback {
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, fallbackOpts.path))
			os.Remove(fallbackOpts.path)
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, fallbackOpts.legacyPath))
			os.Remove(fallbackOpts.legacyPath)
			utils.LogDebug(fmt.Sprintf("Received %v. Deleting (if exists) %v", statusCode, metadataPath))
			os.Remove(metadataPath)
		}

		if fallbackOpts.enable && canUseFallback {
			utils.Log("Unable to fetch secrets from the Doppler API")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackOpts.path, fallbackOpts.legacyPath, fallbackOpts.passphrase, false)
		}
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	if enableCache && statusCode == 304 {
		utils.LogDebug("Using cached secrets from fallback file")
		cache, err := controllers.SecretsCacheFile(fallbackOpts.path, fallbackOpts.passphrase)
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

		if fallbackOpts.enable {
			utils.Log("Unable to parse the Doppler API response")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackOpts.path, fallbackOpts.legacyPath, fallbackOpts.passphrase, false)
		}
		utils.HandleError(err, "Unable to parse API response")
	}

	writeFallbackFile := fallbackOpts.enable && !fallbackOpts.readonly && nameTransformer == nil
	if writeFallbackFile {
		utils.LogDebug("Encrypting secrets")
		encryptedResponse, err := crypto.Encrypt(fallbackOpts.passphrase, response, "base64")
		if err != nil {
			utils.HandleError(err, "Unable to encrypt your secrets. No fallback file has been written.")
		}

		utils.LogDebug(fmt.Sprintf("Writing to fallback file %s", fallbackOpts.path))
		if err := utils.WriteFile(fallbackOpts.path, []byte(encryptedResponse), utils.RestrictedFilePerms()); err != nil {
			utils.Log("Unable to write to fallback file")
			if fallbackOpts.exitOnWriteFailure {
				utils.HandleError(err, "", strings.Join(writeFailureMessage(), "\n"))
			} else {
				utils.LogDebugError(err)
			}
		}

		if enableCache {
			if etag := respHeaders.Get("etag"); etag != "" {
				hash := crypto.Hash(encryptedResponse)

				if err := controllers.WriteMetadataFile(metadataPath, etag, hash); !err.IsNil() {
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

func writeFailureMessage() []string {
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

func parseSecrets(response []byte) (map[string]string, error) {
	secrets := map[string]string{}
	err := json.Unmarshal(response, &secrets)
	return secrets, err
}

// legacyFallbackFile deprecated file path used by early versions of CLI v3
func legacyFallbackFile(project string, config string) string {
	name := fmt.Sprintf("%s:%s", project, config)
	fileName := fmt.Sprintf(".run-%s.json", crypto.Hash(name))
	return filepath.Join(defaultFallbackDir, fileName)
}

// generate the passphrase used for encrypting a secrets file
func getPassphrase(cmd *cobra.Command, flag string, config models.ScopedOptions) string {
	if cmd.Flags().Changed(flag) {
		return cmd.Flag(flag).Value.String()
	}

	if configuration.CanReadEnv {
		passphrase := os.Getenv("DOPPLER_PASSPHRASE")
		if passphrase != "" {
			utils.Log(valueFromEnvironmentNotice("DOPPLER_PASSPHRASE"))
			return passphrase
		}
	}

	if config.EnclaveProject.Value != "" && config.EnclaveConfig.Value != "" {
		return fmt.Sprintf("%s:%s:%s", config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value)
	}

	return config.Token.Value
}

func initFallbackDir(cmd *cobra.Command, config models.ScopedOptions, format models.SecretsFormat, nameTransformer *models.SecretsNameTransformer, secretNames []string, exitOnWriteFailure bool) (string, string) {
	fallbackPath := ""
	legacyFallbackPath := ""
	if cmd.Flags().Changed("fallback") {
		var err error
		fallbackPath, err = utils.GetFilePath(cmd.Flag("fallback").Value.String())
		if err != nil {
			utils.HandleError(err, "Unable to parse --fallback flag")
		}
	} else {
		fallbackFileName := fmt.Sprintf(".secrets-%s.json", controllers.GenerateFallbackFileHash(config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value, format, nameTransformer, secretNames))
		fallbackPath = filepath.Join(defaultFallbackDir, fallbackFileName)
		// TODO remove this when releasing CLI v4 (DPLR-435)
		if config.EnclaveProject.Value != "" && config.EnclaveConfig.Value != "" {
			// save to old path to maintain backwards compatibility
			legacyFallbackPath = legacyFallbackFile(config.EnclaveProject.Value, config.EnclaveConfig.Value)
		}

		if !utils.Exists(defaultFallbackDir) {
			err := os.Mkdir(defaultFallbackDir, 0700)
			if err != nil {
				utils.LogDebug("Unable to create directory for fallback file")
				if exitOnWriteFailure {
					utils.HandleError(err, "Unable to create directory for fallback file", strings.Join(writeFailureMessage(), "\n"))
				}
			}
		}
	}

	if absFallbackPath, err := filepath.Abs(fallbackPath); err == nil {
		fallbackPath = absFallbackPath
	}

	if legacyFallbackPath != "" {
		if absFallbackPath, err := filepath.Abs(legacyFallbackPath); err == nil {
			legacyFallbackPath = absFallbackPath
		}
	}

	return fallbackPath, legacyFallbackPath
}

func getCacheFileETag(metadataPath string, cachePath string) string {
	metadata, Err := controllers.MetadataFile(metadataPath)
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

func init() {
	defaultFallbackDir = filepath.Join(configuration.UserConfigDir, "fallback")
	controllers.DefaultMetadataDir = defaultFallbackDir

	forwardSignals := !isatty.IsTerminal(os.Stdout.Fd())

	runCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	runCmd.RegisterFlagCompletionFunc("project", projectIDsValidArgs)
	runCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	runCmd.RegisterFlagCompletionFunc("config", configNamesValidArgs)
	runCmd.Flags().String("command", "", "command to execute (e.g. \"echo hi\")")
	// note: requires using "--preserve-env=VALUE", doesn't work with "--preserve-env VALUE"
	runCmd.Flags().String("preserve-env", "false", "a comma separated list of secrets for which the existing value from the environment, if any, should take precedence of the Doppler secret value. value must be specified with an equals sign (e.g. --preserve-env=\"FOO,BAR\"). specify \"true\" to give precedence to all existing environment values, however this has potential security implications and should be used at your own risk.")
	// we must specify a default when no value is passed as this flag used to be a boolean
	// https://github.com/spf13/pflag#setting-no-option-default-values-for-flags
	runCmd.Flags().Lookup("preserve-env").NoOptDefVal = "true"
	runCmd.Flags().String("name-transformer", "", fmt.Sprintf("output name transformer. one of %v", validEnvCompatNameTransformersList))
	runCmd.RegisterFlagCompletionFunc("name-transformer", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return models.SecretsEnvCompatNameTransformerTypes, cobra.ShellCompDirectiveDefault
	})
	runCmd.Flags().Duration("dynamic-ttl", 0, "(BETA) dynamic secrets will expire after specified duration, (e.g. '3h', '15m')")
	// fallback flags
	runCmd.Flags().String("fallback", "", "path to the fallback file. encrypted secrets are written to this file after each successful fetch. secrets will be read from this file if subsequent connections are unsuccessful.")
	// TODO rename this to 'fallback-passphrase' in CLI v4 (DPLR-435)
	runCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the fallback file. the default passphrase is computed using your current configuration.")
	runCmd.Flags().Bool("no-cache", false, "disable using the fallback file to speed up fetches. the fallback file is only used when the API indicates that it's still current.")
	runCmd.Flags().Bool("no-fallback", false, "disable reading and writing the fallback file (implies --no-cache)")
	runCmd.Flags().Bool("fallback-readonly", false, "disable modifying the fallback file. secrets can still be read from the file.")
	runCmd.Flags().Bool("fallback-only", false, "read all secrets directly from the fallback file, without contacting Doppler. secrets will not be updated. (implies --fallback-readonly)")
	runCmd.Flags().Bool("no-exit-on-write-failure", false, "do not exit if unable to write the fallback file")
	runCmd.Flags().Bool("forward-signals", forwardSignals, "forward signals to the child process (defaults to false when STDOUT is a TTY)")
	// secrets mount flags
	runCmd.Flags().String("mount", "", "write secrets to an ephemeral file, accessible at DOPPLER_CLI_SECRETS_PATH. when enabled, secrets are NOT injected into the environment")
	runCmd.Flags().String("mount-format", "json", fmt.Sprintf("file format to use. if not specified, will be auto-detected from mount name. one of %v", models.SecretsMountFormats))
	runCmd.RegisterFlagCompletionFunc("mount-format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{projectTemplateFileName}, cobra.ShellCompDirectiveDefault
	})
	runCmd.Flags().String("mount-template", "", "template file to use. secrets will be rendered into this template before mount. see 'doppler secrets substitute' for more info.")
	runCmd.Flags().Int("mount-max-reads", 0, "maximum number of times the mounted secrets file can be read (0 for unlimited)")
	runCmd.Flags().StringSliceVar(&secretsToInclude, "only-secrets", []string{}, "only include the specified secrets")
	runCmd.Flags().Bool("no-exit-on-missing-only-secrets", false, "do not exit on missing secrets via --only-secrets")

	// deprecated
	runCmd.Flags().Bool("silent-exit", false, "disable error output if the supplied command exits non-zero")
	if err := runCmd.Flags().MarkDeprecated("silent-exit", "this behavior is now the default"); err != nil {
		utils.HandleError(err)
	}
	if err := runCmd.Flags().MarkHidden("silent-exit"); err != nil {
		utils.HandleError(err)
	}

	rootCmd.AddCommand(runCmd)

	runCleanCmd.Flags().Duration("max-age", defaultFallbackFileMaxAge, "delete fallback files that exceed this age")
	runCleanCmd.Flags().Bool("dry-run", false, "do not delete anything, print what would have happened")
	runCleanCmd.Flags().Bool("all", false, "delete all fallback files")
	runCmd.AddCommand(runCleanCmd)
}
