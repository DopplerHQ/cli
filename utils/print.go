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
package utils

import (
	"doppler-cli/models"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

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

// PrintLogs print logs
func PrintLogs(logs []models.Log, number int, jsonFlag bool) {
	maxLogs := int(math.Min(float64(len(logs)), float64(number)))

	if jsonFlag {
		PrintJSON(logs[0:maxLogs])
		return
	}

	for _, log := range logs {
		PrintLog(log, false)
	}
}

// PrintLog print log
func PrintLog(log models.Log, jsonFlag bool) {
	if jsonFlag {
		PrintJSON(log)
		return
	}

	dateTime, err := time.Parse(time.RFC3339, log.CreatedAt)

	fmt.Println("Log " + log.ID)
	fmt.Println("User: " + log.User.Name + " <" + log.User.Email + ">")
	if err == nil {
		fmt.Println("Date: " + dateTime.In(time.Local).String())
	}
	fmt.Println("")
	fmt.Println("\t" + log.Text)
	fmt.Println("")
}

// PrintJSON print object as json
func PrintJSON(structure interface{}) {
	resp, err := json.Marshal(structure)
	if err != nil {
		Err(err, "")
	}

	fmt.Println(string(resp))
}
