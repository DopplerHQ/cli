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
package gui

import (
	"context"

	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/cli/pkg/tui/common"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/gocui"
	"github.com/sasha-s/go-deadlock"
)

type Gui struct {
	*common.Common

	g       *gocui.Gui
	cmps    Components
	mutexes Mutexes

	isFetching    bool
	statusMessage string
}

type Mutexes struct {
	SecretViewsMutex *deadlock.Mutex
}

func NewGui(cmn *common.Common) (*Gui, error) {
	gui := &Gui{
		Common: cmn,
		mutexes: Mutexes{
			SecretViewsMutex: &deadlock.Mutex{},
		},
	}

	return gui, nil
}

func (gui *Gui) run() error {
	g, err := gocui.NewGui(gocui.OutputTrue, false, gocui.NORMAL, false, nil)
	if err != nil {
		return err
	}

	gui.g = g
	defer gui.g.Close()

	// Managers are run on every render cycle
	gui.g.SetManager(gocui.ManagerFunc(gui.layout))

	if err := gui.createAllComponents(); err != nil {
		return err
	}

	gui.bindKey("", gocui.KeyCtrlC, gocui.ModNone, func(v *gocui.View) error {
		return gocui.ErrQuit
	})
	gui.bindKey("", 'q', gocui.ModNone, func(v *gocui.View) error {
		return gocui.ErrQuit
	})
	gui.bindKey("", '1', gocui.ModNone, func(v *gocui.View) error {
		return gui.focusComponent(gui.cmps.configs)
	})
	gui.bindKey("", '2', gocui.ModNone, func(v *gocui.View) error {
		return gui.focusComponent(gui.cmps.projects)
	})
	gui.bindKey("", '3', gocui.ModNone, func(v *gocui.View) error {
		return gui.focusComponent(gui.cmps.secrets)
	})
	gui.bindKey("", '/', gocui.ModNone, func(v *gocui.View) error {
		return gui.focusComponent(gui.cmps.secretsFilter)
	})
	gui.bindKey("", '?', gocui.ModNone, func(v *gocui.View) error {
		return gui.focusComponent(gui.cmps.promptHelp)
	})

	g.Highlight = true
	g.SelFgColor = gocui.ColorMagenta
	g.SelFrameColor = gocui.ColorMagenta

	// Set the default component focus
	if configuration.TUIShouldShowIntro() {
		gui.focusComponent(gui.cmps.promptIntro)
	} else {
		gui.focusComponent(gui.cmps.secrets)
	}

	// Fetch the data needed for the initial state of the app
	go gui.load()

	return gui.g.MainLoop()
}

// bindKey must bind a key or panics
func (gui *Gui) bindKey(viewname string, key interface{}, mod gocui.Modifier, handler func(*gocui.View) error) {
	err := gui.g.SetKeybinding(viewname, key, mod, func(g *gocui.Gui, v *gocui.View) error {
		if gui.isFetching && key != gocui.KeyCtrlC {
			return nil
		}
		return handler(v)
	})
	if err != nil {
		panic(err)
	}
}

func (gui *Gui) setIsFetching(isFetching bool) {
	gui.statusMessage = ""
	gui.isFetching = isFetching
	gui.renderAllStateBasedComponents()

	ctx := context.Background()
	if isFetching {
		gui.g.StartTicking(ctx)
	} else {
		ctx.Done()
	}
}

func SafeWithError(f func() error) error {
	panicking := true
	defer func() {
		if panicking && gocui.Screen != nil {
			gocui.Screen.Fini()
		}
	}()

	err := f()

	panicking = false

	return err
}

// recoverScreenOnCrash MUST be deferred in EVERY goroutine that the TUI spawns to ensure that a
// panic doesn't leave the user's terminal in a broken state.
func recoverScreenOnCrash() {
	if r := recover(); r != nil {
		gocui.Screen.Fini()
		panic(r)
	}
}

func (gui *Gui) RunAndHandleError() error {
	return SafeWithError(func() error {
		if err := gui.run(); err != nil {
			switch err {
			case gocui.ErrQuit:
				return nil
			default:
				utils.Print(err.Error())
				return err
			}
		}

		return nil
	})
}
