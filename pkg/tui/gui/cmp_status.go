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

import "github.com/DopplerHQ/gocui"

type StatusComponent struct {
	*BaseComponent
}

var _ Component = &StatusComponent{}

func CreateStatusComponent(gui *Gui) (*StatusComponent, error) {
	cmp := &StatusComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	return cmp, nil
}

func (self *StatusComponent) GetViewName() string { return "Status" }
func (self *StatusComponent) GetTitle() string    { return "Status" }
func (self *StatusComponent) GetFocusable() bool  { return false }

func (self *StatusComponent) Render() error {
	self.GetView().Clear()

	self.GetView().HasLoader = self.gui.isFetching
	if self.gui.isFetching {
		self.GetView().WriteString("Fetching...")
	} else {
		if len(self.gui.statusMessage) > 0 {
			self.GetView().FgColor = gocui.ColorRed
			self.GetView().WriteString(self.gui.statusMessage)
		} else {
			self.GetView().FgColor = gocui.ColorWhite
			self.GetView().WriteString("Ready")
		}
	}

	return nil
}
