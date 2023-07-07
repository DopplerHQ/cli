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

type ConfigsComponent struct {
	*BaseComponent

	selectedIdx int
}

var _ Component = &ConfigsComponent{}

func CreateConfigsComponent(gui *Gui) (*ConfigsComponent, error) {
	cmp := &ConfigsComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	cmp.view.Highlight = true
	cmp.view.SelFgColor = gocui.ColorMagenta
	cmp.view.SelBgColor = gocui.ColorBlack

	gui.bindKey("Configs", 'j', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx + 1)
	})
	gui.bindKey("Configs", 'k', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx - 1)
	})
	gui.bindKey("Configs", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		go gui.selectConfig(cmp.selectedIdx)
		return nil
	})

	return cmp, nil
}

func (self *ConfigsComponent) SelectIdx(idx int) error {
	maxIdx := len(state.Configs()) - 1
	newIdx, err := SelectIdx(self, idx, maxIdx)
	if err != nil {
		return err
	}
	self.selectedIdx = newIdx
	return nil
}

func (self *ConfigsComponent) GetViewName() string { return "Configs" }
func (self *ConfigsComponent) GetTitle() string    { return "Configs (1)" }

func (self *ConfigsComponent) OnFocus() {
	if self.selectedIdx >= len(state.Configs()) {
		self.SelectIdx(0)
	}
}

func (self *ConfigsComponent) Render() error {
	text := ""

	_, activeConf := state.Active()
	for _, conf := range state.Configs() {
		if conf.Name == activeConf {
			text += "* "
		}
		text += conf.Name + "\n"
	}

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
