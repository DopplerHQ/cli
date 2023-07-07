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

func SelectIdx(cmp Component, idx int, maxIdx int) (int, error) {
	if idx < 0 {
		return 0, nil
	}

	if idx > maxIdx {
		return maxIdx, nil
	}

	view := cmp.GetView()

	minVisible := view.OriginY()
	maxVisible := minVisible + view.InnerHeight()

	if idx >= minVisible && idx <= maxVisible {
		if err := view.SetCursor(0, (idx - minVisible)); err != nil {
			return 0, err
		}
	} else if idx < minVisible {
		if err := view.SetOriginY(idx); err != nil {
			return 0, err
		}
		if err := view.SetCursor(0, 0); err != nil {
			return 0, err
		}
	} else {
		if err := view.SetOriginY(idx - view.InnerHeight()); err != nil {
			return 0, err
		}
		if err := view.SetCursor(0, view.InnerHeight()); err != nil {
			return 0, err
		}
	}

	return idx, nil
}
