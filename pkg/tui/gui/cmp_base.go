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

type Component interface {
	GetViewName() string
	GetView() *gocui.View
	GetTitle() string
	GetFocusable() bool

	// Full re-render from State
	Render() error

	OnFocus()
	OnBlur()
}

type BaseComponent struct {
	gui  *Gui
	view *gocui.View
}

func CreateBaseComponent(gui *Gui, cmp Component) (*BaseComponent, error) {
	var err error

	baseCmp := &BaseComponent{gui: gui}

	view, err := gui.createView(cmp)
	if err != nil {
		return nil, err
	}
	baseCmp.view = view

	return baseCmp, nil
}

func (self *BaseComponent) GetView() *gocui.View {
	return self.view
}

func (self *BaseComponent) GetFocusable() bool {
	return true
}

func (self *BaseComponent) OnFocus() {

}

func (self *BaseComponent) OnBlur() {

}

func (self *BaseComponent) Render() error {
	return nil
}
