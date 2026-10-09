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
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"sync"
	"syscall"
	"time"

	"github.com/DopplerHQ/cli/pkg/utils"
)

// signals that ask doppler to stop. one of these arriving while --watch is restarting the child cancels the restart
var terminationSignals = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT}

// runSignalHandler owns signal handling for the lifetime of `doppler run`. It forwards signals to whichever child
// process is current. If a termination signal arrives while --watch is restarting the child, doppler exits as soon as
// no child is running instead of starting a process the user has asked to stop.
type runSignalHandler struct {
	forwardSignals bool
	listenOnce     sync.Once

	// mu also serializes starting a child against handling a signal, so doppler can't exit between creating a
	// child's secrets mount and recording how to clean it up
	mu         sync.Mutex
	child      *exec.Cmd
	cleanup    func()
	restarting bool
	stopSignal os.Signal
}

func newRunSignalHandler(forwardSignals bool) *runSignalHandler {
	return &runSignalHandler{forwardSignals: forwardSignals}
}

// listen starts handling signals. It's deferred until the first child is about to start so that signals received
// before then (e.g. Ctrl-C while fetching secrets) keep their default behavior
func (h *runSignalHandler) listen() {
	h.listenOnce.Do(func() {
		// signal handling logic adapted from aws-vault https://github.com/99designs/aws-vault/
		// buffer generously since we're notified of every signal, including the runtime's frequent SIGURG
		sigChan := make(chan os.Signal, 32)
		signal.Notify(sigChan)
		go func() {
			for sig := range sigChan {
				h.handle(sig)
			}
		}()
	})
}

func (h *runSignalHandler) handle(sig os.Signal) {
	h.mu.Lock()
	child := h.child
	if h.restarting && slices.Contains(terminationSignals, sig) {
		if h.stopSignal == nil {
			utils.LogDebug(fmt.Sprintf("Received %v while restarting process; exiting instead of restarting", sig))
			h.stopSignal = sig
		}
		// with no child running there's nothing to wait for, so don't let slower work (e.g. fetching secrets) delay us
		if child == nil {
			h.exitLocked()
		}
	}
	h.mu.Unlock()

	// When running with a TTY, user-generated signals (like SIGINT) are sent to the entire process group.
	// If we forward the signal, the child process will end up receiving the signal twice.
	if h.forwardSignals && child != nil {
		child.Process.Signal(sig) // #nosec G104
	}
}

// beginRestart marks the start of a --watch restart. The old child keeps receiving forwarded signals until it exits
func (h *runSignalHandler) beginRestart() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.restarting = true
}

// childExited clears the current child once it has exited during a restart, exiting if we've been asked to stop
func (h *runSignalHandler) childExited() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.child = nil
	if h.stopSignal != nil {
		h.exitLocked()
	}
}

// start calls startChild and records the resulting child and its cleanup func, which it also returns
func (h *runSignalHandler) start(startChild func() (*exec.Cmd, func(), error)) (*exec.Cmd, func(), error) {
	h.listen()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.stopSignal != nil {
		h.exitLocked()
	}

	c, cleanup, err := startChild()
	h.cleanup = cleanup
	if err != nil {
		return nil, cleanup, err
	}
	h.child = c
	h.restarting = false
	return c, cleanup, nil
}

// exitLocked cleans up after the most recent child and exits with the stop signal. h.mu must be held
func (h *runSignalHandler) exitLocked() {
	if h.cleanup != nil {
		h.cleanup()
	}
	exitWithSignal(h.stopSignal)
}

// exitWithSignal terminates doppler with sig, as though doppler had never handled it
func exitWithSignal(sig os.Signal) {
	exitCode := 1
	if s, ok := sig.(syscall.Signal); ok {
		exitCode = 128 + int(s)
	}

	// the Go runtime handles an unhandled SIGQUIT by dumping goroutines, and Windows can't signal its own process,
	// so in those cases exit the way a shell reports a process killed by the signal
	if sig != syscall.SIGQUIT && !utils.IsWindows() {
		signal.Reset(sig)
		if p, err := os.FindProcess(os.Getpid()); err == nil {
			p.Signal(sig) // #nosec G104
			// delivery is asynchronous; give it a moment before falling back (e.g. if the signal is ignored)
			time.Sleep(200 * time.Millisecond)
		}
	}

	os.Exit(exitCode)
}
