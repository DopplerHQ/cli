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
)

// Debug whether we're running in debug mode
var Debug = false

// JSON whether to print JSON
var JSON = false

// Log info
func Log(info string) {
	if !JSON {
		fmt.Println(info)
	}
}

// Err prints the error and exits with code 1
func Err(e error, message string) {
	ErrExit(e, message, 1)
}

// ErrExit prints the error and exits with the specified code
func ErrExit(e error, message string, exitCode int) {
	if JSON {
		resp, err := json.Marshal(map[string]string{"error": e.Error()})
		if err != nil {
			panic(err)
		}
		fmt.Println(string(resp))
	} else {
		if message != "" {
			fmt.Println(message)
		}

		fmt.Println("Error:", e)
	}

	if Debug {
		fmt.Println("")
		debug.PrintStack()
	}

	os.Exit(exitCode)
}
