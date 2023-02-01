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
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666)
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
