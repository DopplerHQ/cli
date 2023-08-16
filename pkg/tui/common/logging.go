/*
Copyright © 2023 Doppler <support@doppler.com>

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
package common

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/sirupsen/logrus"
)

func newLogger() *logrus.Entry {
	var log *logrus.Logger
	if utils.DebugTUI {
		log = newDebugLogger()
	} else {
		log = newDiscardedLogger()
	}

	return log.WithFields(logrus.Fields{})
}

func newDebugLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logPath, err := LogPath()
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		log.Fatalf("Unable to log to log file: %v", err)
	}
	logger.SetOutput(file)
	return logger
}

func newDiscardedLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	return logger
}

func LogPath() (string, error) {
	return filepath.Join(configuration.UserConfigDir, "tui.log"), nil
}
