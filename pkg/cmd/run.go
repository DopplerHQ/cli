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

var runCmd = &cobra.Command{
	Use:   "run [command]",
	Short: "Run a command with secrets injected into the environment",
	Long: `Run a command with secrets injected into the environment

To view the CLI's active configuration, run ` + "`doppler configure debug`",
	Example: `doppler run -- YOUR_COMMAND --YOUR-FLAG
doppler run --command "YOUR_COMMAND && YOUR_OTHER_COMMAND"`,
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
		preserveEnv := utils.GetBoolFlag(cmd, "preserve-env")
		forwardSignals := utils.GetBoolFlag(cmd, "forward-signals")
		localConfig := configuration.LocalConfig(cmd)
		dynamicSecretsTTL := utils.GetDurationFlag(cmd, "dynamic-ttl")

		utils.RequireValue("token", localConfig.Token.Value)

		fallbackPath := ""
		legacyFallbackPath := ""
		metadataPath := ""
		if enableFallback {
			fallbackPath, legacyFallbackPath = initFallbackDir(cmd, localConfig, exitOnWriteFailure)
		}
		if enableCache {
			metadataPath = controllers.MetadataFilePath(localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
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

		nameTransformerString := cmd.Flag("name-transformer").Value.String()
		var nameTransformer *models.SecretsNameTransformer
		if nameTransformerString != "" {
			nameTransformer = models.SecretsNameTransformerMap[nameTransformerString]
			if nameTransformer == nil || !nameTransformer.EnvCompat {
				utils.HandleError(fmt.Errorf("invalid name transformer. Valid transformers are %s", validEnvCompatNameTransformersList))
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
		secrets := fetchSecrets(localConfig, enableCache, fallbackOpts, metadataPath, nameTransformer, dynamicSecretsTTL)

		if preserveEnv {
			utils.LogWarning("Ignoring Doppler secrets already defined in the environment due to --preserve-env flag")
		}

		env := os.Environ()
		existingEnvKeys := map[string]bool{}
		for _, envVar := range env {
			// key=value format
			parts := strings.SplitN(envVar, "=", 2)
			key := parts[0]
			existingEnvKeys[key] = true
		}

		excludedKeys := []string{"PATH", "PS1", "HOME"}
		for name, value := range secrets {
			useSecret := true
			for _, excludedKey := range excludedKeys {
				if excludedKey == name {
					useSecret = false
					break
				}
			}

			if useSecret && preserveEnv {
				// skip secret if environment already contains variable w/ same name
				if existingEnvKeys[name] == true {
					utils.LogDebug(fmt.Sprintf("Ignoring Doppler secret %s", name))
					useSecret = false
				}
			}

			if useSecret {
				env = append(env, fmt.Sprintf("%s=%s", name, value))
			}
		}

		exitCode := 0
		var err error

		if cmd.Flags().Changed("command") {
			command := cmd.Flag("command").Value.String()
			exitCode, err = utils.RunCommandString(command, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals)
		} else {
			exitCode, err = utils.RunCommand(args, env, os.Stdin, os.Stdout, os.Stderr, forwardSignals)
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

// fetchSecrets fetches secrets, including all reading and writing of fallback files
func fetchSecrets(localConfig models.ScopedOptions, enableCache bool, fallbackOpts fallbackOptions, metadataPath string, nameTransformer *models.SecretsNameTransformer, dynamicSecretsTTL time.Duration) map[string]string {
	if fallbackOpts.exclusive {
		if !fallbackOpts.enable {
			utils.HandleError(errors.New("Conflict: unable to specify --no-fallback with --fallback-only"))
		}
		if nameTransformer != nil {
			utils.HandleError(errors.New("Conflict: unable to specify --name-transformer with --fallback-only"))
		}
		return readFallbackFile(fallbackOpts.path, fallbackOpts.legacyPath, fallbackOpts.passphrase, false)
	}

	// this scenario likely isn't possible, but just to be safe, disable using cache when there's no metadata file
	enableCache = enableCache && nameTransformer == nil && metadataPath != ""
	etag := ""
	if enableCache {
		etag = getCacheFileETag(metadataPath, fallbackOpts.path)
	}

	statusCode, respHeaders, response, httpErr := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, models.JSON, nameTransformer, etag, dynamicSecretsTTL)
	if !httpErr.IsNil() {
		if fallbackOpts.enable {
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

		// TODO remove this when releasing CLI v4 (DPLR-435)
		if fallbackOpts.legacyPath != "" && localConfig.EnclaveProject.Value != "" && localConfig.EnclaveConfig.Value != "" {
			utils.LogDebug(fmt.Sprintf("Writing to legacy fallback file %s", fallbackOpts.legacyPath))
			if err := utils.WriteFile(fallbackOpts.legacyPath, []byte(encryptedResponse), utils.RestrictedFilePerms()); err != nil {
				utils.Log("Unable to write to legacy fallback file")
				if fallbackOpts.exitOnWriteFailure {
					utils.HandleError(err, "", strings.Join(writeFailureMessage(), "\n"))
				} else {
					utils.LogDebugError(err)
				}
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
	// default to hex for backwards compatibility b/c we didn't always include a prefix
	// TODO remove support for optional prefix when releasing CLI v4 (DPLR-435)
	encoding := "hex"
	if strings.HasPrefix(string(response), crypto.Base64EncodingPrefix) {
		encoding = "base64"
	}
	decryptedSecrets, err := crypto.Decrypt(passphrase, response, encoding)
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

func defaultFallbackFile(token string, project string, config string) string {
	var fileName string
	var name string
	if project == "" && config == "" {
		name = fmt.Sprintf("%s", token)
	} else {
		name = fmt.Sprintf("%s:%s:%s", token, project, config)
	}

	fileName = fmt.Sprintf(".secrets-%s.json", crypto.Hash(name))
	return filepath.Join(defaultFallbackDir, fileName)
}

// generate the passphrase used for encrypting a secrets file
func getPassphrase(cmd *cobra.Command, flag string, config models.ScopedOptions) string {
	if cmd.Flags().Changed(flag) {
		return cmd.Flag(flag).Value.String()
	}

	if config.EnclaveProject.Value != "" && config.EnclaveConfig.Value != "" {
		return fmt.Sprintf("%s:%s:%s", config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value)
	}

	return config.Token.Value
}

func initFallbackDir(cmd *cobra.Command, config models.ScopedOptions, exitOnWriteFailure bool) (string, string) {
	fallbackPath := ""
	legacyFallbackPath := ""
	if cmd.Flags().Changed("fallback") {
		var err error
		fallbackPath, err = utils.GetFilePath(cmd.Flag("fallback").Value.String())
		if err != nil {
			utils.HandleError(err, "Unable to parse --fallback flag")
		}
	} else {
		fallbackPath = defaultFallbackFile(config.Token.Value, config.EnclaveProject.Value, config.EnclaveConfig.Value)
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
	runCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	runCmd.Flags().String("command", "", "command to execute (e.g. \"echo hi\")")
	runCmd.Flags().Bool("preserve-env", false, "ignore any Doppler secrets that are already defined in the environment. this has potential security implications, use at your own risk.")
	runCmd.Flags().String("name-transformer", "", fmt.Sprintf("(BETA) output name transformer. one of %v", validEnvCompatNameTransformersList))
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
