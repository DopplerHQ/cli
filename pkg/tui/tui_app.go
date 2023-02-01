package tui

import (
	"log"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/common"
	"github.com/DopplerHQ/cli/pkg/tui/gui"
)

type App struct {
	*common.Common
	gui *gui.Gui
}

func Start(opts models.ScopedOptions) {
	cmn, err := common.NewCommon(opts)
	if err != nil {
		log.Fatal(err)
	}

	gui, err := gui.NewGui(cmn)
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		Common: cmn,
		gui:    gui,
	}

	app.gui.RunAndHandleError()
}
