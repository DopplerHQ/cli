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
	"encoding/json"
	"fmt"
	"os"
	"runtime/debug"

	"gopkg.in/gookit/color.v1"
)

// Log info to stdout
func Log(info string) {
	silent := Silent || OutputJSON
	if Debug || !silent {
		fmt.Println(info)
	}
}

// LogDebug prints a debug message to stdout
func LogDebug(s string) {
	if Debug {
		fmt.Println(color.Blue.Render("Debug:"), s)
	}
}

// LogError prints an error message to stderr
func LogError(e error) {
	fmt.Fprintln(os.Stderr, color.Red.Render("Doppler Error:"), e)
}

// HandleError prints the error and exits with code 1
func HandleError(e error, messages ...string) {
	ErrExit(e, 1, messages...)
}

// ErrExit prints the error and exits with the specified code
func ErrExit(e error, exitCode int, messages ...string) {
	if OutputJSON {
		resp, err := json.Marshal(map[string]string{"error": e.Error()})
		if err != nil {
			panic(err)
		}
		fmt.Fprintln(os.Stderr, string(resp))
	} else {
		if len(messages) > 0 {
			for _, message := range messages[0:1] {
				fmt.Fprintln(os.Stderr, message)
			}
		}

		LogError(e)

		if len(messages) > 0 {
			for _, message := range messages[1:] {
				fmt.Fprintln(os.Stderr, message)
			}
		}
	}

	if Debug {
		fmt.Fprintln(os.Stderr, color.Red.Render("\nStacktrace:"))
		debug.PrintStack()
	}

	os.Exit(exitCode)
}
