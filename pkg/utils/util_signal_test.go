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
package utils

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"
)

const signalHelperEnv = "DOPPLER_TEST_SIGNAL_HELPER"

// signal tests re-execute the test binary as a helper process so that a SIGINT which isn't handled kills the helper
// instead of the test runner
func TestMain(m *testing.M) {
	switch os.Getenv(signalHelperEnv) {
	case "":
		os.Exit(m.Run())
	case "run-command":
		runCommandHelper()
	case "forward-signals":
		forwardSignalsHelper()
	default:
		panic("unknown signal helper")
	}
}

// interruptSelf sends SIGINT to this process, then exits 0 if it survives
func interruptSelf() {
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		panic(err)
	}
	time.Sleep(2 * time.Second)
	fmt.Println("survived SIGINT")
	os.Exit(0)
}

func runCommandHelper() {
	cmd, err := RunCommand([]string{"true"}, os.Environ(), nil, nil, nil)
	if err != nil {
		panic(err)
	}
	if _, err := WaitCommand(cmd); err != nil {
		panic(err)
	}

	// the subprocess has exited, so SIGINT should now terminate this process
	interruptSelf()
}

func forwardSignalsHelper() {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	cmd, err := RunCommand([]string{"sh", "-c", `trap 'kill $! 2>/dev/null; exit 5' INT; echo ready; sleep 10 >/dev/null & wait`}, os.Environ(), nil, w, os.Stderr)
	if err != nil {
		panic(err)
	}
	w.Close()
	stop := ForwardSignals(cmd, syscall.SIGINT)

	// wait for the child to install its trap
	if line, _ := bufio.NewReader(r).ReadString('\n'); line != "ready\n" {
		panic(fmt.Sprintf("unexpected output from child: %q", line))
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		panic(err)
	}
	exitCode, _ := WaitCommand(cmd)
	fmt.Printf("child exited with %d\n", exitCode)
	stop()

	// forwarding has stopped, so SIGINT should now terminate this process
	interruptSelf()
}

// runSignalHelper runs the named helper and returns its output, failing unless it was killed by SIGINT
func runSignalHelper(t *testing.T, helper string) string {
	t.Helper()
	// the helper inherits our signal dispositions, and an inherited SIG_IGN would mask the behavior under test
	if signal.Ignored(syscall.SIGINT) {
		t.Skip("SIGINT is ignored by this process")
	}

	cmd := exec.Command(os.Args[0]) // #nosec G204
	cmd.Env = append(os.Environ(), signalHelperEnv+"="+helper)
	out, err := cmd.Output()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected helper to be killed by SIGINT, got err=%v; output: %q", err, out)
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() || status.Signal() != syscall.SIGINT {
		t.Fatalf("expected helper to be killed by SIGINT, got %v; output: %q", exitErr, out)
	}
	return string(out)
}

func TestSignalsNotInterceptedAfterWaitCommand(t *testing.T) {
	runSignalHelper(t, "run-command")
}

func TestForwardSignals(t *testing.T) {
	out := runSignalHelper(t, "forward-signals")
	if !strings.Contains(out, "child exited with 5") {
		t.Fatalf("expected SIGINT to be forwarded to the child; output: %q", out)
	}
}
