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

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

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
