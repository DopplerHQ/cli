package gui

import (
	"github.com/DopplerHQ/cli/pkg/tui/gui/state"
	"github.com/DopplerHQ/gocui"
)

type ProjectsComponent struct {
	*BaseComponent

	selectedIdx int
}

var _ Component = &ProjectsComponent{}

func CreateProjectsComponent(gui *Gui) (*ProjectsComponent, error) {
	cmp := &ProjectsComponent{}

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

	cmp.view.Highlight = true
	cmp.view.SelFgColor = gocui.ColorMagenta
	cmp.view.SelBgColor = gocui.ColorBlack

	gui.bindKey("Projects", 'j', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx + 1)
	})
	gui.bindKey("Projects", 'k', gocui.ModNone, func(v *gocui.View) error {
		return cmp.SelectIdx(cmp.selectedIdx - 1)
	})
	gui.bindKey("Projects", gocui.KeyEnter, gocui.ModNone, func(v *gocui.View) error {
		go gui.selectProject(cmp.selectedIdx)
		return nil
	})

	return cmp, nil
}

func (self *ProjectsComponent) SelectIdx(idx int) error {
	maxIdx := len(state.Projects()) - 1
	var err error
	self.selectedIdx, err = SelectIdx(self, idx, maxIdx)
	return err
}

func (self *ProjectsComponent) GetViewName() string { return "Projects" }
func (self *ProjectsComponent) GetTitle() string    { return "Projects (2)" }

func (self *ProjectsComponent) Render() error {
	text := ""

	activeProj, _ := state.Active()
	for _, proj := range state.Projects() {
		if proj.Name == activeProj {
			text += "* "
		}
		text += proj.Name + "\n"
	}

	self.GetView().Clear()
	self.GetView().WriteString(text)

	return nil
}
