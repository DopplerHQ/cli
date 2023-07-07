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
	"github.com/DopplerHQ/gocui"
)

type PromptHelpComponent struct {
	*BaseComponent
}

var _ Component = &PromptHelpComponent{}

func CreatePromptHelpComponent(gui *Gui) (*PromptHelpComponent, error) {
	cmp := &PromptHelpComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	gui.bindKey("PromptHelp", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptHelp.Close()
	})
	gui.bindKey("PromptHelp", gocui.KeyEsc, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptHelp.Close()
	})
	gui.bindKey("PromptHelp", 'q', gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptHelp.Close()
	})

	cmp.GetView().Visible = false

	return cmp, nil
}

func (self *PromptHelpComponent) GetViewName() string { return "PromptHelp" }
func (self *PromptHelpComponent) GetTitle() string    { return "Help" }
func (self *PromptHelpComponent) GetFocusable() bool  { return true }

func (self *PromptHelpComponent) OnFocus() {
	self.Render()
	self.gui.g.SetViewOnTop(self.GetViewName())
	self.GetView().Visible = true
}

func (self *PromptHelpComponent) OnBlur() {
	self.GetView().Visible = false
}

func (self *PromptHelpComponent) Close() error {
	return self.gui.focusComponent(self.gui.cmps.secrets)
}

func (self *PromptHelpComponent) Render() error {
	text := `Global Keybinds:
    1 Focus Configs
    2 Focus Projects
    3 Focus Secrets
    / Focus Filter
    q Exit

Configs / Projects List Keybinds:
    j     Move cursor down
    k     Move cursor up
    Enter Select

Secrets Keybinds:
    j     Move cursor down
    k     Move cursor up
    h / l Toggle between name and value
    J     Scroll current selection down
    K     Scroll current selection up
    e     Enter edit mode
    s     Open save prompt
    a     Add new secret
    d     Delete current secret
    u     Undo changes
    y     Copy current selection to clipboard

Secrets Editing Mode Keybinds:
    Esc Exit editing mode
    Tab Toggle between name and value

Save Prompt Keybinds:
    Enter   Confirm
    Esc / q Cancel

Save Prompt Keybinds:
    Enter   Confirm
    Esc / q Cancel

Filter Keybinds:
    Enter / Esc Stop filtering`

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
