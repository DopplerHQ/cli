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
package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/atotto/clipboard"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/text/transform"
)

// ConfigDir DEPRECATED get configuration directory
func ConfigDir() string {
	// this function is deprecated and should not be used.
	// in testing, node:12-alpine creates the ~/.config directory at some indeterminate point
	// in the build. this means some doppler commands called by docker RUN may use the home
	// directory to store config, while doppler commands called by ENTRYPOINT will use ~/config.
	dir, err := os.UserConfigDir()
	if err != nil {
		HandleError(err, "Unable to determine configuration directory")
	}

	return dir
}

// HomeDir get home directory
func HomeDir() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		HandleError(err, "Unable to determine home directory")
	}

	return dir
}

// ParsePath returns an absolute path, parsing ~ . .. etc
func ParsePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("Path cannot be blank")
	}

	if strings.HasPrefix(path, "~") {
		firstPath := strings.Split(path, string(filepath.Separator))[0]

		if firstPath != "~" {
			username, err := user.Current()
			if err != nil || firstPath != fmt.Sprintf("~%s", username.Username) {
				return "", fmt.Errorf("unable to parse path, please specify an absolute path (e.g. /home/%s)", path[1:])
			}
		}

		path = strings.Replace(path, firstPath, HomeDir(), 1)
	}

	absolutePath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}

	return absolutePath, nil
}

// Exists whether path exists and the user has permission
func Exists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

// Cwd current working directory of user's shell
func Cwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		HandleError(err)
	}
	return cwd
}

// RunCommand runs the specified command
func RunCommand(command []string, env []string, inFile *os.File, outFile io.Writer, errFile io.Writer, forwardSignals bool, onExit func()) (int, error) {
	cmd := exec.Command(command[0], command[1:]...) // #nosec G204
	cmd.Env = env
	cmd.Stdin = inFile
	cmd.Stdout = outFile
	cmd.Stderr = errFile

	return execCommand(cmd, forwardSignals, onExit)
}

// RunCommandString runs the specified command string
func RunCommandString(command string, env []string, inFile *os.File, outFile io.Writer, errFile io.Writer, forwardSignals bool, onExit func()) (int, error) {
	shell := [2]string{"sh", "-c"}
	if IsWindows() {
		shell = [2]string{"cmd", "/C"}
	} else {
		// these shells all support the same options we use for sh
		shells := []string{"/bash", "/dash", "/fish", "/zsh", "/ksh", "/csh", "/tcsh"}
		envShell := os.Getenv("SHELL")
		for _, s := range shells {
			if strings.HasSuffix(envShell, s) {
				shell[0] = envShell
				break
			}
		}
	}
	cmd := exec.Command(shell[0], shell[1], command) // #nosec G204
	cmd.Env = env
	cmd.Stdin = inFile
	cmd.Stdout = outFile
	cmd.Stderr = errFile

	return execCommand(cmd, forwardSignals, onExit)
}

func execCommand(cmd *exec.Cmd, forwardSignals bool, onExit func()) (int, error) {
	if onExit != nil {
		// ensure the onExit handler is called, regardless of how/when we return
		defer onExit()
	}

	// signal handling logic adapted from aws-vault https://github.com/99designs/aws-vault/
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)

	if err := cmd.Start(); err != nil {
		return 1, err
	}

	// handle all signals
	go func() {
		for {
			// When running with a TTY, user-generated signals (like SIGINT) are sent to the entire process group.
			// If we forward the signal, the child process will end up receiving the signal twice.
			if forwardSignals {
				// forward to process
				sig := <-sigChan
				cmd.Process.Signal(sig) // #nosec G104
			} else {
				// ignore
				<-sigChan
			}
		}
	}()

	if err := cmd.Wait(); err != nil {
		// ignore errors
		cmd.Process.Signal(os.Kill) // #nosec G104

		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), exitError
		}

		return 2, err
	}

	waitStatus, ok := cmd.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return 2, fmt.Errorf("Unexpected ProcessState type, expected syscall.WaitStatus, got %T", waitStatus)
	}
	return waitStatus.ExitStatus(), nil
}

// RequireValue throws an error if a value is blank
func RequireValue(name string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		HandleError(fmt.Errorf("you must provide a %s", name))
	}
}

// GetBool parse string into a boolean
func GetBool(value string, def bool) bool {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return def
	}
	return b
}

// GetBoolFlag gets the flag's boolean value
func GetBoolFlag(cmd *cobra.Command, flag string) bool {
	b, err := strconv.ParseBool(cmd.Flag(flag).Value.String())
	if err != nil {
		HandleError(err)
	}
	return b
}

// GetBoolFlagIfChanged gets the flag's boolean value, if specified;
// protects against reading an undefined flag
func GetBoolFlagIfChanged(cmd *cobra.Command, flag string, def bool) bool {
	if !cmd.Flags().Changed(flag) {
		return def
	}

	return GetBoolFlag(cmd, flag)
}

// GetFlagIfChanged gets the flag's value, if specified;
// protects against reading an undefined flag
func GetFlagIfChanged(cmd *cobra.Command, flag string, def string) string {
	if !cmd.Flags().Changed(flag) {
		return def
	}

	return cmd.Flag(flag).Value.String()
}

// GetPathFlagIfChanged gets the flag's path, if specified;
// always returns an absolute path
func GetPathFlagIfChanged(cmd *cobra.Command, flag string, def string) string {
	if !cmd.Flags().Changed(flag) {
		return def
	}

	path, err := ParsePath(cmd.Flag(flag).Value.String())
	if err != nil {
		HandleError(err, "Unable to parse path")
	}
	return path
}

// GetIntFlag gets the flag's int value
func GetIntFlag(cmd *cobra.Command, flag string, bits int) int {
	number, err := strconv.ParseInt(cmd.Flag(flag).Value.String(), 10, bits)
	if err != nil {
		HandleError(err)
	}

	return int(number)
}

// GetDurationFlag gets the flag's duration
func GetDurationFlag(cmd *cobra.Command, flag string) time.Duration {
	value, err := time.ParseDuration(cmd.Flag(flag).Value.String())
	if err != nil {
		HandleError(err)
	}
	return value
}

// GetDurationFlagIfChanged gets the flag's duration, if specified;
// protects against reading an undefined flag
func GetDurationFlagIfChanged(cmd *cobra.Command, flag string, def time.Duration) time.Duration {
	if !cmd.Flags().Changed(flag) {
		return def
	}

	return GetDurationFlag(cmd, flag)
}

// GetFilePath verify a file path and name are provided
func GetFilePath(fullPath string) (string, error) {
	if fullPath == "" {
		return "", errors.New("Invalid file path")
	}

	fullPath, err := ParsePath(fullPath)
	if err != nil {
		return "", errors.New("Invalid file path")
	}

	parsedPath := filepath.Dir(fullPath)
	parsedName := filepath.Base(fullPath)

	isNameValid := (parsedName != ".") && (parsedName != "..") && (parsedName != "/") && (parsedName != string(filepath.Separator))
	if !isNameValid {
		return "", errors.New("Invalid file path")
	}

	return filepath.Join(parsedPath, parsedName), nil
}

// ConfirmationPrompt prompt user to confirm yes/no
func ConfirmationPrompt(message string, defaultValue bool) bool {
	confirm := false
	prompt := &survey.Confirm{
		Message: message,
		Default: defaultValue,
	}

	err := survey.AskOne(prompt, &confirm)
	if err != nil {
		if err == terminal.InterruptErr {
			Log("Exiting")
			os.Exit(1)
		}
		HandleError(err)
	}
	return confirm
}

// SelectPrompt prompt user to select from a list of options
func SelectPrompt(message string, options []string, defaultOption string) string {
	prompt := &survey.Select{
		Message:  message,
		Options:  options,
		PageSize: 25,
	}
	if defaultOption != "" {
		prompt.Default = defaultOption
	}

	selectedProject := ""
	err := survey.AskOne(prompt, &selectedProject)
	if err != nil {
		if err == terminal.InterruptErr {
			Log("Exiting")
			os.Exit(1)
		}
		HandleError(err)
	}

	return selectedProject
}

// CopyToClipboard copies text to the user's clipboard
func CopyToClipboard(text string) error {
	if !clipboard.Unsupported {
		err := clipboard.WriteAll(text)
		if err != nil {
			return err
		}
	}
	return nil
}

// HostOS the host OS
func HostOS() string {
	os := runtime.GOOS

	switch os {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	}

	return os
}

// HostArch the host architecture
func HostArch() string {
	arch := runtime.GOARCH

	switch arch {
	case "amd64":
		return "64-bit"
	case "amd64p32":
	case "386":
		return "32-bit"
	}

	return arch
}

// IsWindows whether the host os is Windows
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS whether the host os is macOS
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// UUID generates a random UUID
func UUID() (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		LogDebug("Unable to generate random UUID")
		return "", err
	}

	return uuid.String(), nil
}

// RedactAuthToken returns a partially-redacted value that's safe for display
func RedactAuthToken(token string) string {
	// ensure there are enough characters or we'll end up giving away the whole token
	if len(token) > 30 {
		return fmt.Sprintf("%s…%s", token[0:10], token[len(token)-5:])
	}

	return "[REDACTED]"
}

// ReplaceTransformer replaces text in a stream
// Taken from https://github.com/icholy/replace/blob/a7e12fe69d82503d82c3f85a9ca3973a11a2085f/replace.go#L12
// See: http://golang.org/x/text/transform
type ReplaceTransformer struct {
	transform.NopResetter

	old, new []byte
	oldlen   int
}

var _ transform.Transformer = (*ReplaceTransformer)(nil)

// BytesReplacer returns a transformer that replaces all instances of old with new.
// Unlike bytes.Replace, empty old values don't match anything.
func BytesReplacer(old, new []byte) ReplaceTransformer {
	return ReplaceTransformer{old: old, new: new, oldlen: len(old)}
}

// Transform implements golang.org/x/text/transform#Transformer
func (t ReplaceTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	var n int
	// don't do anything for empty old string. We're forced to do this because an optimization in
	// transform.String prevents us from generating any output when the src is empty.
	// see: https://github.com/golang/text/blob/master/transform/transform.go#L570-L576
	if t.oldlen == 0 {
		n, err = fullcopy(dst, src)
		return n, n, err
	}
	// replace all instances of old with new
	for {
		i := bytes.Index(src[nSrc:], t.old)
		if i == -1 {
			break
		}
		// copy everything up to the match
		n, err = fullcopy(dst[nDst:], src[nSrc:nSrc+i])
		nSrc += n
		nDst += n
		if err != nil {
			return
		}
		// copy the new value
		n, err = fullcopy(dst[nDst:], t.new)
		if err != nil {
			return
		}
		nDst += n
		nSrc += t.oldlen
	}
	// if we're at the end, tack on any remaining bytes
	if atEOF {
		n, err = fullcopy(dst[nDst:], src[nSrc:])
		nDst += n
		nSrc += n
		return
	}
	// skip everything except the trailing len(r.old) - 1
	// we do this because there could be a match straddling
	// the boundary
	if skip := len(src[nSrc:]) - t.oldlen + 1; skip > 0 {
		n, err = fullcopy(dst[nDst:], src[nSrc:nSrc+skip])
		nSrc += n
		nDst += n
		if err != nil {
			return
		}
	}
	err = transform.ErrShortSrc
	return
}

func fullcopy(dst, src []byte) (n int, err error) {
	n = copy(dst, src)
	if n < len(src) {
		err = transform.ErrShortDst
	}
	return
}
