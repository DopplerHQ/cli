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
	"github.com/DopplerHQ/gocui"
)

type Components struct {
	configs       *ConfigsComponent
	projects      *ProjectsComponent
	secrets       *SecretsComponent
	status        *StatusComponent
	promptSave    *PromptSaveComponent
	promptHelp    *PromptHelpComponent
	promptIntro   *PromptIntroComponent
	secretsFilter *SecretsFilterComponent

	focusable []Component
}

func (gui *Gui) createView(cmp Component) (*gocui.View, error) {
	view, err := gui.g.SetView(cmp.GetViewName(), 0, 0, 10, 10, 0)

	// Unknown view is thrown when a view is created, which is what we're doing here, so
	// we can safely ignore that error.
	if !gocui.IsUnknownView(err) {
		return nil, err
	}

	if cmp.GetFocusable() {
		gui.cmps.focusable = append(gui.cmps.focusable, cmp)
	}

	view.Title = cmp.GetTitle()

	return view, nil
}

func (gui *Gui) createAllComponents() error {
	if configsCmp, err := CreateConfigsComponent(gui); err == nil {
		gui.cmps.configs = configsCmp
	} else {
		return err
	}

	if cmp, err := CreateProjectsComponent(gui); err == nil {
		gui.cmps.projects = cmp
	} else {
		return err
	}

	if cmp, err := CreateSecretsComponent(gui); err == nil {
		gui.cmps.secrets = cmp
	} else {
		return err
	}

	if cmp, err := CreateStatusComponent(gui); err == nil {
		gui.cmps.status = cmp
	} else {
		return err
	}

	if cmp, err := CreatePromptSaveComponent(gui); err == nil {
		gui.cmps.promptSave = cmp
	} else {
		return err
	}

	if cmp, err := CreatePromptHelpComponent(gui); err == nil {
		gui.cmps.promptHelp = cmp
	} else {
		return err
	}

	if cmp, err := CreatePromptIntroComponent(gui); err == nil {
		gui.cmps.promptIntro = cmp
	} else {
		return err
	}

	if cmp, err := CreateSecretsFilterComponent(gui); err == nil {
		gui.cmps.secretsFilter = cmp
	} else {
		return err
	}

	return nil
}

func (gui *Gui) renderAllStateBasedComponents() {
	gui.g.Update(func(*gocui.Gui) error {
		cmps := []Component{
			gui.cmps.configs,
			gui.cmps.projects,
			gui.cmps.secrets,
			gui.cmps.status,
		}
		for _, cmp := range cmps {
			if err := cmp.Render(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (gui *Gui) getCurComponentIdx() (int, bool) {
	curView := gui.g.CurrentView()
	if curView == nil {
		return 0, false
	}

	if curView.ParentView != nil {
		curView = curView.ParentView
	}
	curCmpIdx := -1

	for idx, cmp := range gui.cmps.focusable {
		if cmp.GetView() == curView {
			curCmpIdx = idx
		}
	}

	if curCmpIdx == -1 {
		return 0, false
	}

	return curCmpIdx, true
}

func (gui *Gui) focusComponent(cmp Component) error {
	curCmpIdx, ok := gui.getCurComponentIdx()
	if ok {
		curCmp := gui.cmps.focusable[curCmpIdx]
		curCmp.OnBlur()
	}

	if _, err := gui.g.SetCurrentView(cmp.GetViewName()); err != nil {
		return err
	}

	cmp.OnFocus()
	return nil
}
