/*
Copyright © 2026 Doppler <support@doppler.com>

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
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"
)

const (
	connectedFrame = "event: message\ndata: {\"type\":\"connected\"}\n\n"
	pingFrame      = "event: message\ndata: {\"type\":\"ping\"}\n\n"
	updateFrame    = "event: message\ndata: {\"type\":\"secrets.update\"}\n\n"
)

func TestSSEEventBuffer(t *testing.T) {
	t.Run("partial data emits nothing until the terminator arrives", func(t *testing.T) {
		var buf sseEventBuffer
		if events := buf.append([]byte(connectedFrame[:22])); len(events) != 0 {
			t.Fatalf("expected no events, got %q", events)
		}
		events := buf.append([]byte(connectedFrame[22:]))
		if len(events) != 1 || string(events[0]) != connectedFrame {
			t.Fatalf("expected reassembled frame, got %q", events)
		}
	})

	t.Run("multiple events in one append are emitted individually", func(t *testing.T) {
		var buf sseEventBuffer
		events := buf.append([]byte(pingFrame + updateFrame))
		if len(events) != 2 || string(events[0]) != pingFrame || string(events[1]) != updateFrame {
			t.Fatalf("expected two frames in order, got %q", events)
		}
	})

	t.Run("byte-at-a-time delivery emits each complete event", func(t *testing.T) {
		var buf sseEventBuffer
		var events []string
		for _, b := range []byte(pingFrame + pingFrame) {
			for _, event := range buf.append([]byte{b}) {
				events = append(events, string(event))
			}
		}
		if len(events) != 2 || events[0] != pingFrame || events[1] != pingFrame {
			t.Fatalf("expected two ping frames, got %q", events)
		}
	})

	t.Run("trailing partial data stays buffered", func(t *testing.T) {
		var buf sseEventBuffer
		events := buf.append([]byte(pingFrame + updateFrame[:10]))
		if len(events) != 1 || string(events[0]) != pingFrame {
			t.Fatalf("expected only the complete frame, got %q", events)
		}
		events = buf.append([]byte(updateFrame[10:]))
		if len(events) != 1 || string(events[0]) != updateFrame {
			t.Fatalf("expected buffered frame to complete, got %q", events)
		}
	})
}

// TestPerformSSERequestReassemblesFrames verifies that the SSE reader hands the
// handler complete event frames regardless of how the stream is segmented in
// transit. TCP provides no message boundaries, so a single frame may arrive
// across multiple reads (e.g. the periodic keepalive in issue #536) and
// multiple frames may arrive in a single read.
func TestPerformSSERequestReassemblesFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher := w.(http.Flusher)

		// frame split mid-JSON across two writes
		_, _ = w.Write([]byte(connectedFrame[:22]))
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write([]byte(connectedFrame[22:]))
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)

		// two frames coalesced into a single write
		_, _ = w.Write([]byte(pingFrame + updateFrame))
		flusher.Flush()
	}))
	defer server.Close()

	var mutex sync.Mutex
	var received []string
	handler := func(data []byte) {
		mutex.Lock()
		defer mutex.Unlock()
		received = append(received, string(data))
	}

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	statusCode, _, _ := performSSERequest(req, true, handler)
	if statusCode != 200 {
		t.Fatalf("expected status code 200, got %d", statusCode)
	}

	// the handler is invoked asynchronously; wait for all events to be dispatched
	expected := []string{connectedFrame, pingFrame, updateFrame}
	deadline := time.Now().Add(2 * time.Second)
	for {
		mutex.Lock()
		count := len(received)
		mutex.Unlock()
		if count >= len(expected) || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	mutex.Lock()
	defer mutex.Unlock()
	sort.Strings(received)
	sort.Strings(expected)
	if len(received) != len(expected) {
		t.Fatalf("expected %d complete frames, got %d: %q", len(expected), len(received), received)
	}
	for i := range expected {
		if received[i] != expected[i] {
			t.Errorf("frame mismatch:\nexpected %q\ngot      %q", expected[i], received[i])
		}
	}
}
