package gui

import "github.com/DopplerHQ/gocui"

type Component interface {
	GetViewName() string
	GetView() *gocui.View
	GetTitle() string
	GetFocusable() bool

	// Full re-render from State
	Render() error

	OnFocus()
	OnBlur()
}

type BaseComponent struct {
	gui  *Gui
	view *gocui.View
}

func CreateBaseComponent(gui *Gui, cmp Component) (*BaseComponent, error) {
	var err error

	baseCmp := &BaseComponent{gui: gui}

	baseCmp.view, err = gui.createView(cmp)
	if err != nil {
		return nil, err
	}

	return baseCmp, nil
}

func (self *BaseComponent) GetView() *gocui.View {
	return self.view
}

func (self *BaseComponent) GetFocusable() bool {
	return true
}

func (self *BaseComponent) OnFocus() {

}

func (self *BaseComponent) OnBlur() {

}

func (self *BaseComponent) Render() error {
	return nil
}
