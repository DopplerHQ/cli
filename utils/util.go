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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/mitchellh/go-homedir"
)

// Home get home dir
func Home() string {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return home
}

// Exists whether path exists
func Exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// Cwd current working directory
func Cwd() string {
	cwd, err := os.Executable()
	if err != nil {
		Err(err)
	}
	return path.Dir(cwd)
}

// RunCommand runs the specified command
func RunCommand(command []string, env []string, output bool) error {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = env

	cmdOut, _ := cmd.StdoutPipe()
	cmdErr, _ := cmd.StderrPipe()

	startErr := cmd.Start()
	if startErr != nil {
		fmt.Println(fmt.Sprintf("Error trying to execute command: %s", command))
		Err(startErr)
		return startErr
	}

	if output {
		stdOutput, _ := ioutil.ReadAll(cmdOut)
		errOutput, _ := ioutil.ReadAll(cmdErr)

		fmt.Printf(string(stdOutput))
		fmt.Printf(string(errOutput))
	}

	err := cmd.Wait()
	if err == nil {
		os.Exit(0)
	}

	os.Exit(1)
	return nil
}
