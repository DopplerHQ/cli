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
	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"github.com/DopplerHQ/gocui"
)

type PromptSaveComponent struct {
	*BaseComponent
}

var _ Component = &PromptSaveComponent{}

func CreatePromptSaveComponent(gui *Gui) (*PromptSaveComponent, error) {
	cmp := &PromptSaveComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	gui.bindKey("PromptSave", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptSave.ConfirmSave()
	})
	gui.bindKey("PromptSave", gocui.KeyEsc, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptSave.CancelSave()
	})
	gui.bindKey("PromptSave", 'q', gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptSave.CancelSave()
	})

	cmp.GetView().Visible = false

	return cmp, nil
}

func (self *PromptSaveComponent) GetViewName() string { return "PromptSave" }
func (self *PromptSaveComponent) GetTitle() string    { return "Confirm Changes" }
func (self *PromptSaveComponent) GetFocusable() bool  { return true }

func (self *PromptSaveComponent) OnFocus() {
	self.Render()
	self.gui.g.SetViewOnTop(self.GetViewName())
	self.GetView().Visible = true
}

func (self *PromptSaveComponent) OnBlur() {
	self.GetView().Visible = false
}

func (self *PromptSaveComponent) ConfirmSave() error {
	if len(state.Changes()) == 0 {
		return self.CancelSave()
	}
	go self.gui.saveSecrets(state.Changes())
	return nil
}

func (self *PromptSaveComponent) CancelSave() error {
	return self.gui.focusComponent(self.gui.cmps.secrets)
}

func (self *PromptSaveComponent) Render() error {
	text := ""

	if len(state.Changes()) > 0 {
		text = "The following secrets will be updated: \n\n"
		for _, change := range state.Changes() {
			text += "● " + change.Name + "\n"
		}
	} else {
		text = "There are no changes to save"
	}

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
