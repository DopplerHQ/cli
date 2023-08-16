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
package tui

import (
	"log"
	"os"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/common"
	"github.com/DopplerHQ/cli/pkg/tui/gui"
	"github.com/DopplerHQ/cli/pkg/utils"
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

	if cmn.Opts.EnclaveProject.Value == "" {
		utils.Log("You must run `doppler setup` prior to launching the TUI")
		os.Exit(1)
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
