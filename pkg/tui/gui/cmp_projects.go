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

type ProjectsComponent struct {
	*BaseComponent

	selectedIdx int
}

var _ Component = &ProjectsComponent{}

func CreateProjectsComponent(gui *Gui) (*ProjectsComponent, error) {
	cmp := &ProjectsComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	cmp.view.Highlight = true
	cmp.view.SelFgColor = gocui.ColorMagenta
	cmp.view.SelBgColor = gocui.ColorBlack

	gui.bindKey("Projects", 'j', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx + 1)
	})
	gui.bindKey("Projects", 'k', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx - 1)
	})
	gui.bindKey("Projects", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		go gui.selectProject(cmp.selectedIdx)
		return nil
	})

	return cmp, nil
}

func (self *ProjectsComponent) SelectIdx(idx int) error {
	maxIdx := len(state.Projects()) - 1
	newIdx, err := SelectIdx(self, idx, maxIdx)
	if err != nil {
		return err
	}
	self.selectedIdx = newIdx
	return nil
}

func (self *ProjectsComponent) GetViewName() string { return "Projects" }
func (self *ProjectsComponent) GetTitle() string    { return "Projects (2)" }

func (self *ProjectsComponent) Render() error {
	text := ""

	activeProj, _ := state.Active()
	for _, proj := range state.Projects() {
		if proj.Name == activeProj {
			text += "* "
		}
		text += proj.Name + "\n"
	}

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
