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

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

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
