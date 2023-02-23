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
	var err error

	if gui.cmps.configs, err = CreateConfigsComponent(gui); err != nil {
		return err
	}
	if gui.cmps.projects, err = CreateProjectsComponent(gui); err != nil {
		return err
	}
	if gui.cmps.secrets, err = CreateSecretsComponent(gui); err != nil {
		return err
	}
	if gui.cmps.status, err = CreateStatusComponent(gui); err != nil {
		return err
	}
	if gui.cmps.promptSave, err = CreatePromptSaveComponent(gui); err != nil {
		return err
	}
	if gui.cmps.promptHelp, err = CreatePromptHelpComponent(gui); err != nil {
		return err
	}
	if gui.cmps.promptIntro, err = CreatePromptIntroComponent(gui); err != nil {
		return err
	}
	if gui.cmps.secretsFilter, err = CreateSecretsFilterComponent(gui); err != nil {
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
