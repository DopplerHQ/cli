package common

import (
	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/sirupsen/logrus"
)

// Commonly used things wrapped into one struct for convenience when passing it around
type Common struct {
	Log  *logrus.Entry
	Opts models.ScopedOptions
}

func NewCommon(opts models.ScopedOptions) (*Common, error) {
	return &Common{
		Log:  newLogger(),
		Opts: opts,
	}, nil
}
