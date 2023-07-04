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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/controllers"
	"github.com/DopplerHQ/cli/pkg/crypto"
	"github.com/DopplerHQ/cli/pkg/global"
	"github.com/DopplerHQ/cli/pkg/http"
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var defaultFallbackDir string

const defaultFallbackFileMaxAge = 14 * 24 * time.Hour // 14 days

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
		exitOnMissingIncludedSecrets := !cmd.Flags().Changed("no-exit-on-missing-only-secrets")

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

		fallbackOpts := controllers.FallbackOptions{
			Enable:             enableFallback,
			Path:               fallbackPath,
			LegacyPath:         legacyFallbackPath,
			Readonly:           fallbackReadonly,
			Exclusive:          fallbackOnly,
			ExitOnWriteFailure: exitOnWriteFailure,
			Passphrase:         passphrase,
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

		mountOptions := controllers.MountOptions{
			Enable:   shouldMountFile,
			Format:   mountFormat,
			Path:     mountPath,
			Template: templateBody,
			MaxReads: maxReads,
		}

		watch := cmd.Flags().Changed("watch")

		if watch && fallbackOpts.Exclusive {
			utils.LogWarning("--watch has no effect when used with --fallback-only")
			watch = false
		}

		var c *exec.Cmd
		var cleanupMount func()
		var err error
		var lastSecretsFetch time.Time
		var lastUpdateEvent time.Time
		// used to ensure we only run one process at a time
		var processMutex sync.Mutex
		// used to ensure we only process one event at a time
		var watchMutex sync.Mutex
		// this variable has the potential to be racey, but is made safe by our use of the mutex
		terminatedByWatch := false

		startProcess := func() {
			// ensure we can fetch the new secrets before restarting the process
			secrets := controllers.FetchSecrets(localConfig, enableCache, fallbackOpts, metadataPath, nameTransformer, dynamicSecretsTTL, format, secretsToInclude)
			secretsFetchedAt := time.Now()
			if secretsFetchedAt.After(lastSecretsFetch) {
				lastSecretsFetch = secretsFetchedAt
			}

			controllers.ValidateSecrets(secrets, secretsToInclude, exitOnMissingIncludedSecrets, mountOptions)

			isRestart := c != nil
			// terminate the old process
			if isRestart {
				terminatedByWatch = true

				// killing the process here will cause the cleanup goroutine below to run, thereby unlocking the mutex
				utils.LogDebug(fmt.Sprintf("Sending SIGTERM to process %d", c.Process.Pid))
				c.Process.Signal(syscall.SIGTERM)
				// wait up to 10 sec for the process to exit
				for i := 0; i < 10; i++ {
					if !utils.IsProcessRunning(c.Process) {
						// process has been killed
						break
					}
					if i == 5 {
						utils.LogDebug("Still waiting for process to exit...")
					}
					time.Sleep(time.Second)
				}

				// if the process still hasn't exited, forcefully kill it
				if utils.IsProcessRunning(c.Process) {
					utils.LogDebug("Process has not exited; sending SIGKILL to process")
					if e := c.Process.Kill(); e != nil {
						utils.LogDebugError(e)
					}
				}

				c = nil
			}

			// this lock ensures the old process, if any, has exited before we start a new process
			processMutex.Lock()

			// we could have received a new update event while we were waiting for the previous process to terminate.
			// if so, don't bother starting the process as it'll just be immediately restarted again after fetching the latest secrets
			if lastUpdateEvent.After(secretsFetchedAt) {
				utils.LogDebug("Not starting new process; more recent update event has been received")
				processMutex.Unlock()
				return
			}

			terminatedByWatch = false

			var env []string
			env, cleanupMount = controllers.PrepareSecrets(secrets, os.Environ(), preserveEnv, mountOptions)

			global.WaitGroup.Add(1)

			if isRestart {
				utils.Log("Restarting process")
			}

			// start the process
			c, err = controllers.Run(cmd, args, env, forwardSignals)
			if err != nil {
				defer global.WaitGroup.Done()
				if cleanupMount != nil {
					cleanupMount()
				}
				utils.HandleError(err)
			}

			go func() {
				defer processMutex.Unlock()
				defer global.WaitGroup.Done()

				exitCode, err := utils.WaitCommand(c)

				if cleanupMount != nil {
					cleanupMount()

					// add arbitrary delay before re-mounting secrets to avoid a race-related "broken pipe" when writing to the named pipe
					time.Sleep(100 * time.Millisecond)
				}

				// ignore errors if we were responsible for killing the process
				if !terminatedByWatch {
					if err != nil {
						if strings.HasPrefix(err.Error(), "exec") || strings.HasPrefix(err.Error(), "fork/exec") {
							utils.LogError(err)
						}
						utils.LogDebugError(err)
					}

					os.Exit(exitCode)
				}
			}()
		}

		watchHandler := func(data []byte) {
			event := controllers.ParseWatchEvent(data)
			if event.Type == "" {
				return
			}

			// don't capture analytics for the ping event; it's too noisy
			if event.Type != "ping" {
				controllers.CaptureEvent("WatchDataReceived", map[string]interface{}{"event": event.Type})
			}

			if event.Type == "secrets.update" {
				eventReceived := time.Now()
				if lastUpdateEvent.Before(eventReceived) {
					lastUpdateEvent = eventReceived
				}

				watchMutex.Lock()
				defer watchMutex.Unlock()

				utils.LogDebug(fmt.Sprintf("Received %s event", event.Type))

				// due to the lock, we only process one event at a time, so this event could have come in many seconds ago.
				// it's possible we've already refetched secrets since then, in which case we don't need to re-fetch
				if lastSecretsFetch.After(eventReceived) {
					utils.LogDebug("Ignoring event; newer secrets have already been fetched")
					return
				}

				startProcess()
			} else if event.Type == "connected" {
				utils.LogDebug("Connected to secrets stream")
			} else if event.Type == "ping" {
				// do nothing
			} else {
				utils.Log(fmt.Sprintf("Received unknown event: %s", event.Type))
			}
		}

		startProcess()

		// initiate watch logic after starting the process so that failing to watch just degrades to normal 'run' behavior
		if watch {
			maxAttempts := 10
			attempt := 0
			_ = utils.Retry(maxAttempts, time.Second, func() error {
				attempt = attempt + 1

				statusCode, headers, httpErr := http.WatchSecrets(localConfig.APIHost.Value, utils.GetBool(localConfig.VerifyTLS.Value, true), localConfig.Token.Value, localConfig.EnclaveProject.Value, localConfig.EnclaveConfig.Value, watchHandler)
				if !httpErr.IsNil() {
					e := httpErr.Unwrap()

					msg := "Unable to watch for secrets changes"
					// a 200 is sent as soon as the connection is established, so if it dies after that the status code
					// will still be 200. we should retry these requests
					canRetry := http.IsRetry(statusCode, headers.Get("content-type")) || statusCode == 200
					if canRetry && attempt < maxAttempts {
						msg += ". Will retry"
					}
					if statusCode == 200 {
						// this connection was likely killed due to a timeout, so we can log quietly
						utils.LogDebugError(errors.New(msg))
					} else {
						utils.LogError(errors.New(msg))
					}

					controllers.CaptureEvent("WatchConnectionError", map[string]interface{}{"statusCode": statusCode, "canRetry": canRetry})

					if statusCode != 0 {
						e = fmt.Errorf("%s. Status code: %d", e, statusCode)
					}
					utils.LogDebugError(e)

					if !canRetry {
						return utils.StopRetryError(e)
					}
				}

				return httpErr.Unwrap()
			})
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
					utils.HandleError(err, "Unable to create directory for fallback file", strings.Join(controllers.WriteFailureMessage(), "\n"))
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
	// we only restart the process if it hasn't already exited
	runCmd.Flags().Bool("watch", false, "(BETA) automatically restart the process when secrets change")

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
