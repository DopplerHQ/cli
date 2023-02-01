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

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

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
	var err error
	self.selectedIdx, err = SelectIdx(self, idx, maxIdx)
	return err
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
