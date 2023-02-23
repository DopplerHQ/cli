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
	"math"
	"strings"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/gocui"
	"github.com/jesseduffield/lazycore/pkg/boxlayout"
)

func (gui *Gui) getWindowDimensions() map[string]boxlayout.Dimensions {
	width, height := gui.g.Size()

	sideSectionWeight, mainSectionWeight := gui.getSectionWeights()

	root := &boxlayout.Box{
		Direction: boxlayout.COLUMN,
		Weight:    1,
		Children: []*boxlayout.Box{
			{Direction: boxlayout.ROW, Weight: sideSectionWeight, Children: gui.sideSectionChildren()},
			{Direction: boxlayout.ROW, Weight: mainSectionWeight, Children: gui.mainSectionChildren()},
		},
	}

	return boxlayout.ArrangeWindows(root, 0, 0, width, height)
}

func (gui *Gui) mainSectionChildren() []*boxlayout.Box {
	return []*boxlayout.Box{
		{Window: "Secrets", Weight: 1},
		{Size: 3, Direction: boxlayout.COLUMN, Children: []*boxlayout.Box{
			{Window: "Status", Weight: 2},
			{Window: "SecretsFilter", Weight: 1},
		}},
	}
}

func (gui *Gui) getSectionWeights() (int, int) {
	sidePanelWidthRatio := 0.2
	mainSectionWeight := int(1/sidePanelWidthRatio) - 1
	sideSectionWeight := 1
	return sideSectionWeight, mainSectionWeight
}

func (gui *Gui) sideSectionChildren() []*boxlayout.Box {
	return []*boxlayout.Box{
		{Window: "Configs", Weight: 1},
		{Window: "Projects", Weight: 1},
	}
}

func (gui *Gui) layout(g *gocui.Gui) error {
	viewDimensions := gui.getWindowDimensions()

	// We assume that the view has already been created.
	setViewFromDimensions := func(viewName string) (*gocui.View, error) {
		dimensionsObj, ok := viewDimensions[viewName]
		if !ok {
			return nil, fmt.Errorf("could not find dimensions for view %s", viewName)
		}

		view, err := g.View(viewName)
		if err != nil {
			return nil, err
		}

		frameOffset := 1
		if view.Frame {
			frameOffset = 0
		}
		_, err = g.SetView(
			viewName,
			dimensionsObj.X0-frameOffset,
			dimensionsObj.Y0-frameOffset,
			dimensionsObj.X1+frameOffset,
			dimensionsObj.Y1+frameOffset,
			0,
		)

		return view, err
	}

	setSecretViewSizes := func() error {
		gui.mutexes.SecretViewsMutex.Lock()
		defer gui.mutexes.SecretViewsMutex.Unlock()

		secX0, secY0, secX1, secY1 := gui.cmps.secrets.view.Dimensions()
		secWidth := gui.cmps.secrets.view.InnerWidth()

		svmNameX := secX0 + 1
		svmNameWidth := int(math.Floor(float64(secWidth) * 0.3))
		svmValueX := secX0 + svmNameWidth + 2

		asvm := gui.cmps.secrets.activeSVM

		// The Y positions of the secret views are dependent on how much we need to offset to ensure
		// that the active SVM is always fully within the bounds of the secrets view
		if asvm != nil {
			// We use the value view because the name view will always be one line tall
			_, asvmY0, _, asvmY1 := asvm.valueView.Dimensions()
			if asvmY1 >= secY1 {
				gui.cmps.secrets.scrollDelta += (asvmY1 - secY1) + 1
			} else if asvmY0 <= secY0 {
				gui.cmps.secrets.scrollDelta += (asvmY0 - secY0) - 1
			}
		}
		curY := secY0 - gui.cmps.secrets.scrollDelta + 1

		for _, svm := range gui.cmps.secrets.visibleSVMs() {
			numLines := len(strings.Split(svm.valueView.TextArea.GetContent(), "\n"))
			valueHeight := utils.Clamp(numLines, 1, 8)

			_, err := g.SetView(svm.nameView.Name(), svmNameX, curY, svmNameX+svmNameWidth, curY+2, 0)
			if err != nil {
				return err
			}

			_, err = g.SetView(svm.valueView.Name(), svmValueX, curY, secX1-1, curY+1+valueHeight, 0)
			if err != nil {
				return err
			}

			if svm == asvm {
				// If possible, we want to pin the Y origin at the top (which improves the behavior of the
				// textarea as it's resizing). gocui is smart enough to adjust the origin down if the cursor
				// is out of bounds of the view size.
				svm.valueView.SetOriginY(0)
				svm.valueView.RenderTextArea()
			}

			curY += valueHeight + 2
		}

		return nil
	}

	centerPrompt := func(viewName string, width int, height int) error {
		winWidth, winHeight := gui.g.Size()

		_, err := g.SetView(
			viewName,
			winWidth/2-width/2,
			winHeight/2-height/2,
			winWidth/2+width/2,
			winHeight/2+int(math.Ceil(float64(height)/2.0)),
			0,
		)
		if err != nil {
			return err
		}
		return nil
	}

	setPromptSaveSize := func() error {
		width := 80
		height := utils.Clamp(gui.cmps.promptSave.GetView().LinesHeight(), 2, 10)
		return centerPrompt(gui.cmps.promptSave.GetView().Name(), width, height)
	}

	setPromptHelpSize := func() error {
		width := 80
		height := gui.cmps.promptHelp.GetView().LinesHeight() + 1
		return centerPrompt(gui.cmps.promptHelp.GetView().Name(), width, height)
	}

	setPromptIntroSize := func() error {
		width := 80
		height := gui.cmps.promptIntro.GetView().LinesHeight() + 1
		return centerPrompt(gui.cmps.promptIntro.GetView().Name(), width, height)
	}

	if _, err := setViewFromDimensions("Configs"); err != nil {
		return err
	}
	if _, err := setViewFromDimensions("Projects"); err != nil {
		return err
	}
	if _, err := setViewFromDimensions("Secrets"); err != nil {
		return err
	}
	if _, err := setViewFromDimensions("Status"); err != nil {
		return err
	}
	if _, err := setViewFromDimensions("SecretsFilter"); err != nil {
		return err
	}

	if err := setSecretViewSizes(); err != nil {
		return err
	}

	if err := setPromptSaveSize(); err != nil {
		return err
	}
	if err := setPromptHelpSize(); err != nil {
		return err
	}
	if err := setPromptIntroSize(); err != nil {
		return err
	}

	return nil
}
