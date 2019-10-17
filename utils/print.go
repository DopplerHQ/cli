package utils

import (
	"os"

	"github.com/olekukonko/tablewriter"
)

// PrintTable prints table
func PrintTable(headers []string, rows [][]string) {
	// TODO doesn't handle multi line secrets well
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader(headers)
	table.AppendBulk(rows)

	table.Render()
}
