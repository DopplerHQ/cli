package gui

import "github.com/DopplerHQ/gocui"

type StatusComponent struct {
	*BaseComponent
}

var _ Component = &StatusComponent{}

func CreateStatusComponent(gui *Gui) (*StatusComponent, error) {
	cmp := &StatusComponent{}

	var err error
	if cmp.BaseComponent, err = CreateBaseComponent(gui, cmp); err != nil {
		return nil, err
	}

	return cmp, nil
}

func (self *StatusComponent) GetViewName() string { return "Status" }
func (self *StatusComponent) GetTitle() string    { return "Status" }
func (self *StatusComponent) GetFocusable() bool  { return false }

func (self *StatusComponent) Render() error {
	self.GetView().Clear()

	self.GetView().HasLoader = self.gui.isFetching
	if self.gui.isFetching {
		self.GetView().WriteString("Fetching...")
	} else {
		if len(self.gui.statusMessage) > 0 {
			self.GetView().FgColor = gocui.ColorRed
			self.GetView().WriteString(self.gui.statusMessage)
		} else {
			self.GetView().FgColor = gocui.ColorWhite
			self.GetView().WriteString("Ready")
		}
	}

	return nil
}
