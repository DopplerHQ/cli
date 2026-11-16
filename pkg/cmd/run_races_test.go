//go:build !windows
// +build !windows

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

// These tests run a doppler binary built with the race detector through the parts of `doppler run --watch` that share
// state between goroutines, and fail if it reports a data race. The secrets mount's race is covered in pkg/controllers.
package cmd_test

import (
	"strings"
	"syscall"
	"testing"
	"time"
)

func assertNoDataRaces(t *testing.T, p *dopplerProcess) {
	t.Helper()
	if strings.Contains(p.output(), "WARNING: DATA RACE") {
		t.Fatalf("the race detector reported a data race\n%s", p.describe())
	}
}

// Each event from the watch stream is handled in its own goroutine. Updates that arrive while a restart is in progress
// are handled concurrently with it and with each other.
func TestWatchConcurrentEventsHaveNoDataRaces(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, watch: true, raceDetector: true, traps: slowShutdownTraps})
	p.waitForEvent(t, "started 1")

	triggerSecretsUpdate(t, api, p)
	p.waitForEvent(t, "TERMSTART 1")
	// more updates while the first child shuts down. they're spaced out so each is a separate read from the stream
	for i := 0; i < 3; i++ {
		time.Sleep(50 * time.Millisecond)
		triggerSecretsUpdate(t, api, p)
	}

	// the first restart is superseded by the newer events, and one of them starts the next child with fresh secrets
	p.waitForEvent(t, "started 3")
	p.signal(t, syscall.SIGTERM)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 7)
	assertNoDataRaces(t, p)
}

// When the watch stream drops, the connection retry loop runs alongside the event handlers from the old connection,
// and the "connected" event on the new connection refetches secrets in case an update was missed.
func TestWatchReconnectHasNoDataRaces(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, watch: true, raceDetector: true, traps: exitTraps})
	p.waitForEvent(t, "started 1")
	p.waitForOutput(t, "Connected to secrets stream", 1)

	api.dropWatchConnection(t)

	// after reconnecting, doppler refetches secrets and restarts the child
	p.waitForOutput(t, "Connected to secrets stream", 2)
	p.waitForEvent(t, "started 2")
	p.signal(t, syscall.SIGTERM)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 7)
	assertNoDataRaces(t, p)
}
