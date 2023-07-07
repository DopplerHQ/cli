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
	"math"

	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"github.com/DopplerHQ/gocui"
)

type SecretsFilterComponent struct {
	*BaseComponent
}

var _ Component = &SecretsFilterComponent{}

func CreateSecretsFilterComponent(gui *Gui) (*SecretsFilterComponent, error) {
	cmp := &SecretsFilterComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	gui.bindKey("SecretsFilter", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.secretsFilter.Finish()
	})
	gui.bindKey("SecretsFilter", gocui.KeyEsc, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.secretsFilter.Finish()
	})

	cmp.GetView().Editable = true
	cmp.GetView().Editor = gocui.EditorFunc(gui.SecretsFilterEditor)

	return cmp, nil
}

func (self *SecretsFilterComponent) GetViewName() string { return "SecretsFilter" }
func (self *SecretsFilterComponent) GetTitle() string    { return "Filter (/)" }
func (self *SecretsFilterComponent) GetFocusable() bool  { return true }

func (self *SecretsFilterComponent) Finish() error {
	if err := self.gui.focusComponent(self.gui.cmps.secrets); err != nil {
		return err
	}

	return self.gui.cmps.secrets.SelectSVM(0, true)
}

func (gui *Gui) SecretsFilterEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	rendered := gocui.DefaultEditor.Edit(v, key, ch, mod)
	if rendered {
		state.SetFilter(gui.cmps.secretsFilter.GetView().Buffer())
		for _, svm := range gui.cmps.secrets.secretVMs {
			svm.ApplyFilter()
		}

		// As we filter, we want to make sure that we're pinning to the top of the secrets view
		gui.cmps.secrets.scrollDelta = math.MaxInt
		gui.layout(gui.g)
		gui.cmps.secrets.SetActiveSVM(0)
	}
	return rendered
}
