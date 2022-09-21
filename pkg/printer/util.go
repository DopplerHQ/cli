/*
Copyright © 2021 Doppler <support@doppler.com>

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
package printer

import "math"

const colWidthBuffer = 3

// numColumns determine the max number of columns
func numColumns(rows [][]string) int {
	numCols := 0
	for _, row := range rows {
		numCols = int(math.Max(float64(numCols), float64(len(row))))
	}
	return numCols
}

// maxColWidths determine the max width of each column
func maxColWidths(rows [][]string, numCols int) []int {
	lengths := make([]int, numCols)
	for _, row := range rows {
		for i, col := range row {
			lengths[i] = int(math.Max(float64(lengths[i]), float64(len(col))))
		}
	}
	return lengths
}

// optimalColWidths determine the optimal width of each column
func optimalColWidths(maxColWidths []int, numCols int) []int {
	colWidths := make([]int, numCols)
	bufferPool := 0

	for i, maxWidth := range maxColWidths {
		// apply a static buffer to account for table decoration between columns
		initialWidth := (maxTableWidth / numCols) - colWidthBuffer
		colWidth := initialWidth

		if maxWidth < initialWidth {
			colWidth = maxWidth
			reclaimedWidth := initialWidth - colWidth
			bufferPool = bufferPool + reclaimedWidth
		} else if maxWidth > initialWidth {
			dispensedWidth := int(math.Min(float64(bufferPool), float64(maxWidth-initialWidth)))
			colWidth = colWidth + dispensedWidth
			bufferPool = bufferPool - dispensedWidth
		}

		colWidths[i] = colWidth
	}

	return colWidths
}
