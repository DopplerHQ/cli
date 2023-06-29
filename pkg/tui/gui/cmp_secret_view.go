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
	"fmt"
	"strings"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"github.com/DopplerHQ/gocui"
)

type SecretViewModel struct {
	originalName       interface{}
	originalValue      interface{}
	originalVisibility string
	nameView           *gocui.View
	valueView          *gocui.View
	isTouched          bool
	isDirty            bool
	shouldDelete       bool
}

var generateId func() string

func init() {
	curId := 0
	generateId = func() string {
		curId++
		return fmt.Sprint(curId)
	}
}

func (svm *SecretViewModel) ToChangeRequest() models.ChangeRequest {
	cr := models.ChangeRequest{
		OriginalName: svm.originalName,
		Name:         svm.nameView.TextArea.GetContent(),
		Value:        svm.valueView.TextArea.GetContent(),
		ShouldDelete: svm.shouldDelete,
	}

	if svm.originalVisibility != "restricted" {
		cr.OriginalValue = svm.originalValue
	}

	return cr
}

func (svm *SecretViewModel) ShouldSubmit() bool {
	return svm.isDirty && (len(svm.nameView.TextArea.GetContent()) > 0 || (svm.originalName != nil && len(svm.originalName.(string)) > 0))
}

func CreateSecretViewModel(gui *Gui, secret *state.Secret) (*SecretViewModel, error) {
	id := generateId()

	nameView, err := gui.g.SetView("SVM:Name:"+id, 0, 1, 1000, 1, 0)
	nameView.ParentView = gui.cmps.secrets.view
	nameView.ConstrainContentsToParent = true
	if !gocui.IsUnknownView(err) {
		return nil, err
	}

	valueView, err := gui.g.SetView("SVM:Value:"+id, 0, 1, 1000, 1, 0)
	valueView.ParentView = gui.cmps.secrets.view
	valueView.ConstrainContentsToParent = true
	if !gocui.IsUnknownView(err) {
		return nil, err
	}

	nameView.Editor = gocui.EditorFunc(gui.SecretNameEditor)
	nameView.TextArea.Clear()
	nameView.TextArea.TypeString(secret.Name)
	nameView.TextArea.SetCursor2D(0, 0)
	nameView.RenderTextArea()

	valueView.Editor = gocui.EditorFunc(gui.SecretValueEditor)
	valueView.TextArea.Clear()
	if secret.Visibility == "restricted" {
		valueView.TextArea.TypeString("[RESTRICTED]")
	} else {
		valueView.TextArea.TypeString(secret.Value)
	}
	valueView.TextArea.SetCursor2D(0, 0)
	valueView.RenderTextArea()

	svm := &SecretViewModel{
		originalName:       secret.Name,
		originalValue:      secret.Value,
		originalVisibility: secret.Visibility,
		nameView:           nameView,
		valueView:          valueView,
	}

	svm.UpdateViewState()
	svm.ApplyFilter()

	return svm, nil
}

func (gui *Gui) SecretNameEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	switch {
	case key == gocui.KeyEnter:
		gui.cmps.secrets.FinishEditingCurrentField()
		return false
	case key == gocui.KeySpace:
		key = '_'
		ch = '_'
	case (ch > 64 && ch < 91) || (ch > 47 && ch < 58):
		break
	case ch > 96 && ch < 123:
		ch = ch - 32
	default:
		ch = '_'
	}

	rendered := gocui.DefaultEditor.Edit(v, key, ch, mod)
	if rendered {
		gui.cmps.secrets.OnCurrentSVMChanged()
	}
	return rendered
}

func (gui *Gui) SecretValueEditor(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) bool {
	switch {
	case key == gocui.KeyLF:
		key = gocui.KeyEnter
	}

	rendered := gocui.DefaultEditor.Edit(v, key, ch, mod)
	if rendered {
		gui.cmps.secrets.OnCurrentSVMChanged()
	}
	return rendered
}

func (self *SecretViewModel) UpdateViewState() {
	nameChanged := self.nameView.TextArea.GetContent() != self.originalName
	valueChanged := (self.originalVisibility != "restricted" && self.valueView.TextArea.GetContent() != self.originalValue) || (self.originalVisibility == "restricted" && self.isTouched)

	self.isDirty = self.originalName == "" || nameChanged || valueChanged || self.shouldDelete

	if self.shouldDelete {
		self.nameView.FrameColor = gocui.ColorRed
		self.valueView.FrameColor = gocui.ColorRed
		self.nameView.FgColor = gocui.ColorRed
		self.valueView.FgColor = gocui.ColorRed
	} else if self.isDirty {
		self.nameView.FrameColor = gocui.ColorYellow
		self.valueView.FrameColor = gocui.ColorYellow
		self.nameView.FgColor = gocui.ColorYellow
		self.valueView.FgColor = gocui.ColorYellow
	} else {
		self.nameView.FrameColor = gocui.ColorDefault
		self.valueView.FrameColor = gocui.ColorDefault
		self.nameView.FgColor = gocui.ColorDefault
		self.valueView.FgColor = gocui.ColorDefault
	}
}

func (self *SecretViewModel) ApplyFilter() {
	filter := state.Filter()
	shouldShow := self.isDirty || len(filter) == 0 || strings.Index(self.nameView.TextArea.GetContent(), strings.ToUpper(filter)) >= 0
	self.nameView.Visible = shouldShow
	self.valueView.Visible = shouldShow
}
