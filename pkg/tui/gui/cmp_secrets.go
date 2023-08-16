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
	"errors"
	"fmt"
	"strings"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/gocui"
)

type SecretsComponent struct {
	*BaseComponent

	scrollDelta         int
	activeSVM           *SecretViewModel
	svmsForSecretsSetAt int64
	secretVMs           []*SecretViewModel
}

var _ Component = &SecretsComponent{}

func CreateSecretsComponent(gui *Gui) (*SecretsComponent, error) {
	cmp := &SecretsComponent{}

	baseCmp, err := CreateBaseComponent(gui, cmp)
	if err != nil {
		return nil, err
	}
	cmp.BaseComponent = baseCmp

	gui.bindKey("Secrets", 'j', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectDelta(1)
	})
	gui.bindKey("Secrets", 'k', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectDelta(-1)
	})
	gui.bindKey("Secrets", 'h', gocui.ModNone, func(v *gocui.View) error {
		return cmp.ToggleNameValue()
	})
	gui.bindKey("Secrets", 'l', gocui.ModNone, func(v *gocui.View) error {
		return cmp.ToggleNameValue()
	})
	gui.bindKey("Secrets", gocui.KeyTab, gocui.ModNone, func(v *gocui.View) error {
		return cmp.ToggleNameValue()
	})
	gui.bindKey("Secrets", gocui.KeyBacktab, gocui.ModNone, func(v *gocui.View) error {
		return cmp.ToggleNameValue()
	})
	gui.bindKey("Secrets", 'J', gocui.ModNone, func(v *gocui.View) error {
		gui.g.CurrentView().TextArea.MoveCursorDown()
		gui.g.CurrentView().ScrollDown(1)
		return nil
	})
	gui.bindKey("Secrets", 'K', gocui.ModNone, func(v *gocui.View) error {
		gui.g.CurrentView().TextArea.MoveCursorUp()
		gui.g.CurrentView().ScrollDown(-1)
		return nil
	})
	gui.bindKey("Secrets", 'e', gocui.ModNone, func(v *gocui.View) error {
		cmp.EditCurrentField()
		return nil
	})
	gui.bindKey("Secrets", 'y', gocui.ModNone, func(v *gocui.View) error {
		cmp.YankCurrentField()
		return nil
	})
	gui.bindKey("Secrets", 'a', gocui.ModNone, func(v *gocui.View) error {
		return cmp.AppendSVM()
	})
	gui.bindKey("Secrets", 'd', gocui.ModNone, func(v *gocui.View) error {
		return cmp.DeleteSVM()
	})
	gui.bindKey("Secrets", 'u', gocui.ModNone, func(v *gocui.View) error {
		cmp.UndoChanges()
		return nil
	})
	gui.bindKey("Secrets", 's', gocui.ModNone, func(v *gocui.View) error {
		return cmp.PromptSave()
	})
	gui.bindKey("Secrets", gocui.KeyEsc, gocui.ModNone, func(v *gocui.View) error {
		return gui.cmps.secrets.FinishEditingCurrentField()
	})

	return cmp, nil
}

func (self *SecretsComponent) GetViewName() string { return "Secrets" }
func (self *SecretsComponent) GetTitle() string    { return "Secrets (3)" }

func (self *SecretsComponent) visibleSVMs() []*SecretViewModel {
	var vis []*SecretViewModel
	for _, svm := range self.secretVMs {
		if svm.nameView.Visible {
			vis = append(vis, svm)
		}
	}
	return vis
}

func (self *SecretsComponent) SetActiveSVM(idx int) {
	visibleSVMs := self.visibleSVMs()
	if idx < len(visibleSVMs) {
		self.activeSVM = visibleSVMs[idx]
	}
}

func (self *SecretsComponent) SelectSVM(idx int, forceFocusName bool) error {
	focusName := forceFocusName || strings.Index(self.gui.g.CurrentView().Name(), "SVM:Name:") == 0
	visibleSVMs := self.visibleSVMs()
	if idx >= len(visibleSVMs) {
		self.activeSVM = nil
		return nil
	}
	svmToFocus := visibleSVMs[idx]

	var viewToFocus string
	if focusName {
		viewToFocus = svmToFocus.nameView.Name()
	} else {
		viewToFocus = svmToFocus.valueView.Name()
	}

	if _, err := self.gui.g.SetCurrentView(viewToFocus); err != nil {
		return err
	}

	self.activeSVM = svmToFocus
	return nil
}

func (self *SecretsComponent) SelectDelta(delta int) error {
	curIdx := -1
	visibleSVMs := self.visibleSVMs()
	for idx, svm := range visibleSVMs {
		if self.activeSVM == svm {
			curIdx = idx
			break
		}
	}

	if curIdx == -1 {
		return nil
	}

	idxToFocus := utils.Clamp(curIdx+delta, 0, len(visibleSVMs)-1)
	return self.SelectSVM(idxToFocus, false)
}

func (self *SecretsComponent) OnBlur() {
	self.view.TitleColor = gocui.ColorWhite
}

func (self *SecretsComponent) OnFocus() {
	self.view.TitleColor = gocui.ColorMagenta

	if self.activeSVM != nil {
		toFocus := self.activeSVM.nameView
		self.gui.g.SetCurrentView(toFocus.Name())
	}
}

func (self *SecretsComponent) createSVMs() error {
	if state.SecretsSetAt() == self.svmsForSecretsSetAt {
		return nil
	}

	self.gui.mutexes.SecretViewsMutex.Lock()
	defer self.gui.mutexes.SecretViewsMutex.Unlock()

	self.svmsForSecretsSetAt = state.SecretsSetAt()

	for _, oldView := range self.secretVMs {
		self.gui.g.DeleteView(oldView.nameView.Name())
		self.gui.g.DeleteView(oldView.valueView.Name())
	}

	self.scrollDelta = 0
	self.activeSVM = nil
	self.secretVMs = make([]*SecretViewModel, len(state.Secrets()))

	for idx, secret := range state.Secrets() {
		secret := secret
		var err error
		if self.secretVMs[idx], err = CreateSecretViewModel(self.gui, &secret); err != nil {
			return err
		}
	}

	visibleSVMs := self.visibleSVMs()
	if len(visibleSVMs) > 0 {
		self.activeSVM = visibleSVMs[0]
	}

	curViewName := self.gui.g.CurrentView().Name()
	isSVMFocused := strings.Index(curViewName, "SVM:") == 0
	if isSVMFocused || curViewName == self.GetViewName() {
		// We want to focus on the newly created SVM if we were focused on the secrets component
		self.OnFocus()
	}

	return nil
}

func (self *SecretsComponent) ToggleNameValue() error {
	if len(self.visibleSVMs()) == 0 {
		return nil
	}

	isNameFocused := strings.Index(self.gui.g.CurrentView().Name(), "SVM:Name:") == 0
	isEditing := self.gui.g.CurrentView().Editable

	if isEditing {
		if err := self.FinishEditingCurrentField(); err != nil {
			return err
		}
	}

	var err error
	if isNameFocused {
		_, err = self.gui.g.SetCurrentView(self.activeSVM.valueView.Name())
	} else {
		_, err = self.gui.g.SetCurrentView(self.activeSVM.nameView.Name())
	}

	if err != nil {
		return err
	}

	if isEditing {
		self.EditCurrentField()
	}

	return nil
}

func (self *SecretsComponent) AppendSVM() error {
	newSVM, err := CreateSecretViewModel(self.gui, &state.Secret{})
	if err != nil {
		return err
	}
	newSVM.originalName = nil

	self.gui.mutexes.SecretViewsMutex.Lock()
	self.secretVMs = append(self.secretVMs, newSVM)
	self.gui.mutexes.SecretViewsMutex.Unlock()

	// We need to position the new SVM before we can select it
	self.gui.layout(self.gui.g)

	visibleSVMs := self.visibleSVMs()
	if err = self.SelectSVM(len(visibleSVMs)-1, true); err != nil {
		return err
	}

	self.EditCurrentField()
	return nil
}

func (self *SecretsComponent) DeleteSVM() error {
	if self.activeSVM == nil {
		return nil
	}

	if self.activeSVM.originalName == nil {
		self.gui.mutexes.SecretViewsMutex.Lock()
		defer self.gui.mutexes.SecretViewsMutex.Unlock()

		curIdx := -1
		for idx, svm := range self.secretVMs {
			if self.activeSVM == svm {
				curIdx = idx
				break
			}
		}

		curVisibleIdx := -1
		for idx, svm := range self.visibleSVMs() {
			if self.activeSVM == svm {
				curVisibleIdx = idx
				break
			}
		}

		if curIdx == -1 || curVisibleIdx == -1 {
			return errors.New("Attempted to delete but couldn't find active SVM")
		}

		self.secretVMs = append(self.secretVMs[:curIdx], self.secretVMs[curIdx+1:]...)
		if err := self.gui.g.DeleteView(self.activeSVM.nameView.Name()); err != nil {
			return err
		}
		if err := self.gui.g.DeleteView(self.activeSVM.valueView.Name()); err != nil {
			return err
		}

		idxToFocus := utils.Max(curVisibleIdx-1, 0)
		if err := self.SelectSVM(idxToFocus, true); err != nil {
			return err
		}
	} else {
		self.activeSVM.shouldDelete = true
		self.OnCurrentSVMChanged()
	}

	return nil
}

func (self *SecretsComponent) UndoChanges() {
	if self.activeSVM == nil {
		return
	}

	self.activeSVM.shouldDelete = false
	self.activeSVM.isTouched = false

	self.activeSVM.nameView.TextArea.Clear()
	self.activeSVM.nameView.TextArea.TypeString(fmt.Sprint(self.activeSVM.originalName))
	self.activeSVM.nameView.TextArea.SetCursor2D(0, 0)
	self.activeSVM.nameView.RenderTextArea()

	self.activeSVM.valueView.TextArea.Clear()
	if self.activeSVM.originalVisibility == "restricted" {
		self.activeSVM.valueView.TextArea.TypeString("[RESTRICTED]")
	} else {
		self.activeSVM.valueView.TextArea.TypeString(fmt.Sprint(self.activeSVM.originalValue))
	}
	self.activeSVM.valueView.TextArea.SetCursor2D(0, 0)
	self.activeSVM.valueView.RenderTextArea()

	self.OnCurrentSVMChanged()
}

func (self *SecretsComponent) OnCurrentSVMChanged() {
	self.activeSVM.UpdateViewState()
}

func (self *SecretsComponent) EditCurrentField() {
	if self.activeSVM == nil {
		return
	}

	self.activeSVM.isTouched = true

	isValueFocused := strings.Index(self.gui.g.CurrentView().Name(), "SVM:Value:") == 0
	if isValueFocused && self.activeSVM.originalVisibility == "restricted" {
		self.gui.g.CurrentView().TextArea.Clear()
		self.gui.cmps.secrets.OnCurrentSVMChanged()
	}

	self.gui.g.Cursor = true
	self.gui.g.CurrentView().Editable = true
	self.gui.g.CurrentView().ParentView.Editable = true
	self.gui.g.CurrentView().TextArea.SetCursorAtEnd()
	self.gui.g.CurrentView().RenderTextArea()
}

func (self *SecretsComponent) YankCurrentField() error {
	if self.activeSVM == nil {
		return nil
	}

	return utils.CopyToClipboard(self.gui.g.CurrentView().TextArea.GetContent())
}

func (self *SecretsComponent) FinishEditingCurrentField() error {
	if self.activeSVM == nil {
		return errors.New("Attempted to finish editing but no active SVM exists")
	}

	self.gui.g.Cursor = false
	self.gui.g.CurrentView().Editable = false
	self.gui.g.CurrentView().ParentView.Editable = false
	self.gui.g.CurrentView().SetCursor(0, 0)

	return nil
}

func (self *SecretsComponent) PromptSave() error {
	var changes []models.ChangeRequest
	for _, svm := range self.secretVMs {
		if svm.ShouldSubmit() {
			changes = append(changes, svm.ToChangeRequest())
		}
	}

	state.SetChanges(changes)
	return self.gui.focusComponent(self.gui.cmps.promptSave)
}

func (self *SecretsComponent) Render() error {
	if err := self.createSVMs(); err != nil {
		return err
	}

	if len(state.Projects()) > 0 && len(state.Configs()) > 0 {
		curProj, curConf := state.Active()
		self.view.Title = fmt.Sprintf("Secrets (3) [%s / %s]", curProj, curConf)
	}

	return nil
}
