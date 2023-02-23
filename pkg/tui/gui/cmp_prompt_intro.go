package gui

import (
	"github.com/DopplerHQ/cli/pkg/configuration"
	"github.com/DopplerHQ/gocui"
)

type PromptIntroComponent struct {
	*BaseComponent
}

var _ Component = &PromptIntroComponent{}

func CreatePromptIntroComponent(gui *Gui) (*PromptIntroComponent, error) {
	cmp := &PromptIntroComponent{}

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

	gui.bindKey("PromptIntro", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptIntro.Close()
	})
	gui.bindKey("PromptIntro", gocui.KeyEsc, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptIntro.Close()
	})
	gui.bindKey("PromptIntro", 'q', gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.promptIntro.Close()
	})

	cmp.GetView().Visible = false

	return cmp, nil
}

func (self *PromptIntroComponent) GetViewName() string { return "PromptIntro" }
func (self *PromptIntroComponent) GetTitle() string    { return "Welcome" }
func (self *PromptIntroComponent) GetFocusable() bool  { return true }

func (self *PromptIntroComponent) OnFocus() {
	self.Render()
	self.gui.g.SetViewOnTop(self.GetViewName())
	self.GetView().Visible = true
}

func (self *PromptIntroComponent) OnBlur() {
	self.GetView().Visible = false
}

func (self *PromptIntroComponent) Close() error {
	self.gui.Log.Debug("write")
	configuration.TUIMarkIntroSeen()
	self.gui.Log.Debug("wrote")
	return self.gui.focusComponent(self.gui.cmps.secrets)
}

func (self *PromptIntroComponent) Render() error {
	text := `Welcome to the beta version of the Doppler TUI!

To get started, close this window with Escape and then
press ? to view a list of keybindings and supported operations.

We'd love your feedback! Please report any bugs and feature
requests to our CLI repository at:

https://github.com/DopplerHQ/cli`

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
