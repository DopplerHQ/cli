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
