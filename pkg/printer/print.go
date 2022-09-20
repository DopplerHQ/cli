/*
Copyright © 2019 Doppler <support@doppler.com>

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

import (
	"fmt"
	"os"
	"sort"

	"github.com/DopplerHQ/cli/pkg/models"
	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
	goVersion "github.com/hashicorp/go-version"
	"github.com/jedib0t/go-pretty/table"
	"github.com/jedib0t/go-pretty/text"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/gookit/color.v1"
)

type tableOptions struct {
	Title           string
	ShowBorder      bool
	SeparateHeader  bool
	SeparateColumns bool
}

var maxTableWidth = 80

func init() {
	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		utils.LogDebugError(err)
		utils.LogDebug("Unable to determine terminal size")
	} else if w > 0 {
		maxTableWidth = w - 1
	}
}

// TableOptions customize table display
func TableOptions() tableOptions {
	return tableOptions{ShowBorder: true, SeparateHeader: true, SeparateColumns: true}
}

// Table print table
func Table(headers []string, rows [][]string, options tableOptions) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleLight)

	t.SetTitle(options.Title)
	t.Style().Options.DrawBorder = options.ShowBorder
	t.Style().Options.SeparateHeader = options.SeparateHeader
	t.Style().Options.SeparateColumns = options.SeparateColumns

	tableHeaders := table.Row{}
	for _, header := range headers {
		tableHeaders = append(tableHeaders, header)
	}
	t.AppendHeader(tableHeaders)

	numCols := numColumns(rows)
	maxColWidths := maxColWidths(rows, numCols)
	colWidths := optimalColWidths(maxColWidths, numCols)

	for _, row := range rows {
		tableRow := table.Row{}
		for i, val := range row {
			tableRow = append(tableRow, text.WrapText(val, colWidths[i]))
		}
		t.AppendRow(tableRow)
	}

	t.Render()
}

// ChangeLog print change log
func ChangeLog(changes map[string]models.ChangeLog, max int, jsonFlag bool) {
	if jsonFlag {
		JSON(changes)
		return
	}

	var versionsRaw []string
	for v := range changes {
		versionsRaw = append(versionsRaw, v)
	}

	// sort versions so we print the X most recent
	versions := make([]*goVersion.Version, len(versionsRaw))
	for i, raw := range versionsRaw {
		v, _ := goVersion.NewVersion(raw)
		versions[i] = v
	}
	sort.Sort(sort.Reverse(goVersion.Collection(versions)))

	for i, v := range versions {
		if i >= max {
			break
		}
		if i != 0 {
			fmt.Println("")
		}

		vString := version.Normalize(v.String())
		fmt.Println(color.Cyan.Sprintf("CLI %s", vString))
		cl := changes[vString]
		for _, change := range cl.Changes {
			fmt.Println(fmt.Sprintf("· %s", change))
		}
	}
}
