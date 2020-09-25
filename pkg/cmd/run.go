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
	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/gookit/color.v1"
)

var defaultFallbackDir string
var defaultFallbackFileMaxAge = 72 * time.Hour

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
			return errors.New("requires at least 1 arg(s), only received 0")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		enableFallback := !utils.GetBoolFlag(cmd, "no-fallback")
		fallbackReadonly := utils.GetBoolFlag(cmd, "fallback-readonly")
		fallbackOnly := utils.GetBoolFlag(cmd, "fallback-only")
		exitOnWriteFailure := !utils.GetBoolFlag(cmd, "no-exit-on-write-failure")
		silentExit := utils.GetBoolFlag(cmd, "silent-exit")
		preserveEnv := utils.GetBoolFlag(cmd, "preserve-env")
		localConfig := configuration.LocalConfig(cmd)

		utils.RequireValue("token", localConfig.Token.Value)

		fallbackPath := ""
		legacyFallbackPath := ""
		if cmd.Flags().Changed("fallback") {
			var err error
			fallbackPath, err = utils.GetFilePath(cmd.Flag("fallback").Value.String())
			if err != nil {
				utils.HandleError(err, "Unable to parse --fallback flag")
			}
		} else {
			fallbackPath = defaultFallbackFile(localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			// TODO remove this when releasing CLI v4 (DPLR-435)
			if localConfig.EnclaveProject.Value != "" && localConfig.EnclaveConfig.Value != "" {
				// save to old path to maintain backwards compatibility
				legacyFallbackPath = legacyFallbackFile(localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			}

			if enableFallback && !utils.Exists(defaultFallbackDir) {
				err := os.Mkdir(defaultFallbackDir, 0700)
				if err != nil && exitOnWriteFailure {
					utils.HandleError(err, "Unable to create directory for fallback file", strings.Join(writeFailureMessage(), "\n"))
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

		var passphrase string
		if cmd.Flags().Changed("passphrase") {
			passphrase = cmd.Flag("passphrase").Value.String()
		} else {
			// using only the token is sufficient for Service Tokens. it's insufficient for CLI tokens, which require a project and config
			passphrase = fmt.Sprintf("%s", localConfig.Token.Value)
			if localConfig.EnclaveProject.Value != "" && localConfig.EnclaveConfig.Value != "" {
				passphrase = fmt.Sprintf("%s:%s:%s", passphrase, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value)
			}
		}

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

		secrets := fetchSecrets(localConfig, enableFallback, fallbackPath, legacyFallbackPath, fallbackReadonly, fallbackOnly, exitOnWriteFailure, passphrase)

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
			exitCode, err = utils.RunCommandString(command, env, os.Stdin, os.Stdout, os.Stderr)
		} else {
			exitCode, err = utils.RunCommand(args, env, os.Stdin, os.Stdout, os.Stderr)
		}

		if err != nil || exitCode != 0 {
			if silentExit {
				os.Exit(exitCode)
			}
			utils.ErrExit(err, exitCode)
		}
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

		utils.LogDebug(fmt.Sprintf("Using fallback directory %s", defaultFallbackDir))

		if _, err := os.Stat(defaultFallbackDir); err != nil {
			if os.IsNotExist(err) {
				utils.LogDebug("Fallback directory does not exist")
				utils.Log("Nothing to clean")
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

			validUntil := entry.ModTime().Add(maxAge)
			if validUntil.Before(now) {
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
			utils.Log(fmt.Sprintf("%s %d fallback file\n", action, deleted))
		} else {
			utils.Log(fmt.Sprintf("%s %d fallback files\n", action, deleted))
		}
	},
}

func fetchSecrets(localConfig models.ScopedOptions, enableFallback bool, fallbackPath string, legacyFallbackPath string, fallbackReadonly bool, fallbackOnly bool, exitOnWriteFailure bool, passphrase string) map[string]string {
	fetchSecrets := !(enableFallback && fallbackOnly)
	if !fetchSecrets {
		return readFallbackFile(fallbackPath, legacyFallbackPath, passphrase)
	}

	response, httpErr := http.DownloadSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, true)
	if httpErr != (http.Error{}) {
		if enableFallback {
			utils.Log("Unable to fetch secrets from the Doppler API")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackPath, legacyFallbackPath, passphrase)
		}
		utils.HandleError(httpErr.Unwrap(), httpErr.Message)
	}

	// ensure the response can be parsed before proceeding
	secrets, err := parseSecrets(response)
	if err != nil {
		if enableFallback {
			utils.Log("Unable to parse the Doppler API response")
			utils.LogError(httpErr.Unwrap())
			return readFallbackFile(fallbackPath, legacyFallbackPath, passphrase)
		}
		utils.HandleError(err, "Unable to parse API response")
	}

	writeFallbackFile := enableFallback && !fallbackReadonly
	if writeFallbackFile {
		utils.LogDebug("Encrypting secrets")
		encryptedResponse, err := crypto.Encrypt(passphrase, response)
		if err != nil {
			utils.HandleError(err, "Unable to encrypt your secrets. No fallback file has been written.")
		}

		utils.LogDebug(fmt.Sprintf("Writing to fallback file %s", fallbackPath))
		if err := utils.WriteFile(fallbackPath, []byte(encryptedResponse), 0400); err != nil {
			utils.Log("Unable to write to fallback file")
			if exitOnWriteFailure {
				utils.HandleError(err, "", strings.Join(writeFailureMessage(), "\n"))
			} else {
				utils.LogError(err)
			}
		}

		// TODO remove this when releasing CLI v4 (DPLR-435)
		if localConfig.EnclaveProject.Value != "" && localConfig.EnclaveConfig.Value != "" {
			utils.LogDebug(fmt.Sprintf("Writing to legacy fallback file %s", legacyFallbackPath))
			if err := utils.WriteFile(legacyFallbackPath, []byte(encryptedResponse), 0400); err != nil {
				utils.Log("Unable to write to legacy fallback file")
				if exitOnWriteFailure {
					utils.HandleError(err, "", strings.Join(writeFailureMessage(), "\n"))
				} else {
					utils.LogError(err)
				}
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

func readFallbackFile(path string, legacyPath string, passphrase string) map[string]string {
	utils.Log("Reading secrets from fallback file")
	utils.LogDebug(fmt.Sprintf("Using fallback file %s", path))

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// attempt to read from the legacy path, in case the fallback file was created with an older version of the CLI
			// TODO remove this when releasing CLI v4 (DPLR-435)
			if legacyPath != "" {
				return readFallbackFile(legacyPath, "", passphrase)
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

func init() {
	defaultFallbackDir = filepath.Join(configuration.UserConfigDir, "fallback")

	runCmd.Flags().StringP("project", "p", "", "project (e.g. backend)")
	runCmd.Flags().StringP("config", "c", "", "config (e.g. dev)")
	runCmd.Flags().String("fallback", "", "path to the fallback file.write secrets to this file after connecting to Doppler. secrets will be read from this file if subsequent connections are unsuccessful.")
	runCmd.Flags().String("passphrase", "", "passphrase to use for encrypting the fallback file. by default the passphrase is computed using your current configuration.")
	runCmd.Flags().String("command", "", "command to execute (e.g. \"echo hi\")")
	runCmd.Flags().Bool("preserve-env", false, "ignore any Doppler secrets that are already defined in the environment. this has potential security implications, use at your own risk.")
	runCmd.Flags().Bool("no-fallback", false, "disable reading and writing the fallback file")
	runCmd.Flags().Bool("fallback-readonly", false, "disable modifying the fallback file. secrets can still be read from the file.")
	runCmd.Flags().Bool("fallback-only", false, "read all secrets directly from the fallback file, without contacting Doppler. secrets will not be updated. (implies --fallback-readonly)")
	runCmd.Flags().Bool("no-exit-on-write-failure", false, "do not exit if unable to write the fallback file")
	runCmd.Flags().Bool("silent-exit", false, "disable error output if the supplied command exits non-zero")
	rootCmd.AddCommand(runCmd)

	runCleanCmd.Flags().Duration("max-age", defaultFallbackFileMaxAge, "delete fallback files that exceed this age")
	runCleanCmd.Flags().Bool("dry-run", false, "do not delete anything, print what would have happened")
	runCmd.AddCommand(runCleanCmd)
}
