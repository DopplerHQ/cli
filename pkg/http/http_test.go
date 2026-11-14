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
package http

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPerformSSERequestFrames(t *testing.T) {
	connected := "event: message\ndata: {\"type\":\"connected\"}\n\n"
	ping := "event: message\ndata: {\"type\":\"ping\"}\n\n"
	large := "event: message\ndata: {\"type\":\"secrets.update\",\"padding\":\"" + strings.Repeat("x", 2048) + "\"}\n\n"
	for _, tc := range []struct {
		name string
		body string
		want []string
	}{
		{"combined", connected + ping, []string{connected, ping}},
		{"larger than read buffer", large, []string{large}},
		{"CRLF", strings.ReplaceAll(connected+ping, "\n", "\r\n"), []string{connected, ping}},
		{"incomplete final event", connected + "event: message\ndata: {", []string{connected}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)
			events := make(chan string, 100)
			status, headers, err := performSSERequest(req, true, func(data []byte) { events <- string(data) })
			require.True(t, errors.Is(err, io.EOF))
			require.Equal(t, http.StatusOK, status)
			require.Equal(t, "text/event-stream", headers.Get("Content-Type"))
			var got []string
			timer := time.NewTimer(time.Second)
			defer timer.Stop()
		collect:
			for {
				select {
				case event := <-events:
					got = append(got, event)
				case <-timer.C:
					break collect
				}
			}
			sort.Strings(got)
			sort.Strings(tc.want)
			require.Equal(t, tc.want, got)
		})
	}
}
