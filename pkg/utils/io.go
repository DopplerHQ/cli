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
package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// RestrictedFilePerms perms used for creating restrictied files meant to be accessible only to the user
func RestrictedFilePerms() os.FileMode {
	// windows disallows overwriting an existing file with 0400 perms
	if IsWindows() {
		return 0600
	}

	return 0400
}

// WriteFile atomically writes data to a file named by filename.
func WriteFile(filename string, data []byte, perm os.FileMode) error {
	temp := fmt.Sprintf("%s.%s", filename, RandomBase64String(8))

	// write to a unique temp file first before performing an atomic move to the actual file name
	// this prevents a race condition between multiple CLIs reading/writing the same file
	LogDebug(fmt.Sprintf("Writing to temp file %s", temp))
	if err := ioutil.WriteFile(temp, data, os.FileMode(perm)); err != nil {
		return err
	}

	LogDebug(fmt.Sprintf("Renaming temp file to %s", filename))
	if err := os.Rename(temp, filename); err != nil {
		// clean up temp file
		_ = os.Remove(temp)
		return err
	}

	return nil
}

// WriteTempFile writes data to a unique temp file and returns the file name
func WriteTempFile(name string, data []byte, perm os.FileMode) (string, error) {
	// create hidden file in user's home dir to ensure no other users have write access
	tmpFile, err := ioutil.TempFile(HomeDir(), fmt.Sprintf(".%s.", name))
	if err != nil {
		return "", err
	}

	tmpFileName := tmpFile.Name()
	LogDebug(fmt.Sprintf("Writing to temp file %s", tmpFileName))
	if _, err := tmpFile.Write(data); err != nil {
		return tmpFileName, err
	}

	if err := tmpFile.Close(); err != nil {
		return tmpFileName, err
	}

	if err := os.Chmod(tmpFileName, perm); err != nil {
		return tmpFileName, err
	}

	return tmpFileName, nil
}

func HasDataOnStdIn() (bool, error) {
	stat, e := os.Stdin.Stat()
	if e != nil {
		LogDebugError(e)
		return false, errors.New("Unable to stat stdin")
	}

	hasData := (stat.Mode() & os.ModeCharDevice) == 0
	return hasData, nil
}

func GetStdIn() (*string, error) {
	// read from stdin
	hasData, e := HasDataOnStdIn()
	if e != nil {
		return nil, e
	}

	if !hasData {
		return nil, nil
	}

	var input []string
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if ok := scanner.Scan(); !ok {
			if e := scanner.Err(); e != nil {
				return nil, e
			}

			break
		}

		s := scanner.Text()
		input = append(input, s)
	}

	s := strings.Join(input, "\n")
	return &s, nil
}
