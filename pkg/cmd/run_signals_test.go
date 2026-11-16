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

// These tests exercise `doppler run` signal handling end to end: they build the CLI, point it at a mock API,
// and send real signals to the doppler process (or to its whole process group, which is what a TTY does on Ctrl-C).
// The child is a small sh script that appends to an events file whenever it starts or receives a signal.
package cmd_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

var dopplerBinary string

// buildDir holds binaries built by the tests
var buildDir string

// dopplerRaceBinary returns a doppler binary built with the race detector, building it on first use since that's
// slow. DOPPLER_TEST_RACE_BINARY uses a prebuilt one instead
var dopplerRaceBinary = sync.OnceValues(func() (string, error) {
	if binary := os.Getenv("DOPPLER_TEST_RACE_BINARY"); binary != "" {
		return binary, nil
	}
	binary := filepath.Join(buildDir, "doppler-race")
	out, err := exec.Command("go", "build", "-race", "-o", binary, "github.com/DopplerHQ/cli").CombinedOutput() // #nosec G204
	if err != nil {
		return "", fmt.Errorf("%v: %s", err, out)
	}
	return binary, nil
})

// openTTY returns the slave side of a new pseudo-terminal. It's only implemented on some OSes
var openTTY func(t *testing.T) *os.File

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "doppler-signal-tests")
	if err != nil {
		panic(err)
	}
	buildDir = dir
	// DOPPLER_TEST_BINARY runs the tests against a prebuilt binary, e.g. to compare behavior with an older version
	dopplerBinary = os.Getenv("DOPPLER_TEST_BINARY")
	if dopplerBinary == "" {
		dopplerBinary = filepath.Join(dir, "doppler")
		build := exec.Command("go", "build", "-o", dopplerBinary, "github.com/DopplerHQ/cli") // #nosec G204
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "unable to build doppler binary (set DOPPLER_TEST_BINARY to use a prebuilt one): %v\n", err) // nosemgrep: semgrep_configs.prohibit-print
			os.Exit(1)
		}
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// mockAPI serves the endpoints used by `doppler run`. Each secrets download returns {"GEN":"<n>"}, where n counts
// downloads, so every child process can tell which generation it is.
type mockAPI struct {
	server    *httptest.Server
	downloads atomic.Int32
	// when non-zero, downloads from this one onward block until holdDownloads is closed
	holdFrom      int32
	holdDownloads chan struct{}
	releaseOnce   sync.Once
	watchEvents   chan string
}

func newMockAPI(t *testing.T, holdFrom int32) *mockAPI {
	api := &mockAPI{
		holdFrom:      holdFrom,
		holdDownloads: make(chan struct{}),
		watchEvents:   make(chan string),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/configs/config/secrets/download", func(w http.ResponseWriter, r *http.Request) {
		n := api.downloads.Add(1)
		if api.holdFrom != 0 && n >= api.holdFrom {
			select {
			case <-api.holdDownloads:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"GEN":"%d"}`, n)
	})
	mux.HandleFunc("/v3/configs/config/secrets/watch", func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "event: message\ndata: {\"type\":\"connected\"}\n\n")
		flusher.Flush()
		for {
			select {
			case event := <-api.watchEvents:
				if event == dropConnection {
					return
				}
				fmt.Fprintf(w, "event: message\ndata: {\"type\":\"%s\"}\n\n", event)
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	api.server = httptest.NewServer(mux)

	t.Cleanup(func() {
		api.releaseDownloads()
		api.server.CloseClientConnections()
		api.server.Close()
	})
	return api
}

// sent on watchEvents to end the watch stream, which makes doppler reconnect
const dropConnection = "drop-connection"

func (api *mockAPI) dropWatchConnection(t *testing.T) {
	t.Helper()
	select {
	case api.watchEvents <- dropConnection:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out dropping the watch connection")
	}
}

func (api *mockAPI) releaseDownloads() {
	api.releaseOnce.Do(func() { close(api.holdDownloads) })
}

// waitForDownload waits until doppler has requested secrets at least n times
func (api *mockAPI) waitForDownload(t *testing.T, n int32) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for api.downloads.Load() < n {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for doppler to request secrets %d time(s)", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// triggerSecretsUpdate sends a secrets.update event once doppler has processed the "connected" event. The CLI parses
// one event per read from the stream, so sending both at once could merge them into a single unparseable read
func triggerSecretsUpdate(t *testing.T, api *mockAPI, p *dopplerProcess) {
	t.Helper()
	p.waitForOutput(t, "Connected to secrets stream", 1)
	select {
	case api.watchEvents <- "secrets.update":
	case <-p.done:
		// doppler exited; the test's own assertions will report whether that was expected
	case <-time.After(10 * time.Second):
		t.Fatal("timed out sending secrets.update event")
	}
}

type runOptions struct {
	forwardSignals bool
	// don't pass --forward-signals, so doppler uses its default
	defaultForwarding bool
	// connect doppler's stdout to a TTY, which changes the --forward-signals default
	ttyStdout bool
	// pass the script as `-- /bin/sh -c <script>` rather than with --command
	argsForm bool
	watch    bool
	mount    bool
	// run a doppler binary built with the race detector
	raceDetector bool
	// body of the child's sh script. the prelude defines `log` and sets GEN, and the script is followed by an
	// interruptible wait. $! is the pid of that wait's background sleep
	traps string
	// replaces the default interruptible wait at the end of the script
	tail string
}

type dopplerProcess struct {
	cmd       *exec.Cmd
	stateDir  string
	mountPath string
	outPath   string
	done      chan struct{}
}

const scriptPrelude = `log() { echo "$*" >> "$STATE/events"; }
if [ -n "$DOPPLER_CLI_SECRETS_PATH" ]; then GEN=$(tr -dc 0-9 < "$DOPPLER_CLI_SECRETS_PATH"); fi
`

const defaultTail = `log "started $GEN"
sleep 60 >/dev/null 2>&1 &
wait
`

func startDoppler(t *testing.T, api *mockAPI, opts runOptions) *dopplerProcess {
	t.Helper()
	if signal.Ignored(syscall.SIGINT) {
		// doppler would inherit SIG_IGN for SIGINT, which masks the behavior under test
		t.Skip("SIGINT is ignored by this process")
	}

	stateDir := t.TempDir()
	p := &dopplerProcess{
		stateDir: stateDir,
		outPath:  filepath.Join(stateDir, "doppler.log"),
		done:     make(chan struct{}),
	}

	tail := opts.tail
	if tail == "" {
		tail = defaultTail
	}
	script := scriptPrelude + opts.traps + "\n" + tail

	args := []string{"run",
		"--api-host", api.server.URL,
		"--token", "dp.st.test",
		"--project", "test", "--config", "test",
		"--no-fallback", "--no-liveness-ping", "--no-check-version",
	}
	if !opts.defaultForwarding {
		args = append(args, fmt.Sprintf("--forward-signals=%t", opts.forwardSignals))
	}
	if opts.watch {
		// --debug logs when the watch stream connects, which triggerSecretsUpdate waits for
		args = append(args, "--watch", "--debug")
	}
	if opts.mount {
		p.mountPath = filepath.Join(stateDir, "secrets.json")
		args = append(args, "--mount", p.mountPath)
	}
	if opts.argsForm {
		args = append(args, "--", "/bin/sh", "-c", script)
	} else {
		args = append(args, "--command", script)
	}

	// give doppler a clean environment so the developer's own Doppler config can't leak in
	var env []string
	for _, kv := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(kv, "=", 2)[0])
		if strings.HasPrefix(key, "DOPPLER_") || strings.HasSuffix(key, "_PROXY") || key == "SHELL" || key == "HOME" {
			continue
		}
		env = append(env, kv)
	}
	env = append(env,
		"HOME="+stateDir,
		"DOPPLER_CONFIG_DIR="+filepath.Join(stateDir, ".doppler"),
		"DOPPLER_ENABLE_VERSION_CHECK=false",
		"SHELL=/bin/sh",
		"STATE="+stateDir,
	)

	out, err := os.Create(p.outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	binary := dopplerBinary
	if opts.raceDetector {
		var err error
		if binary, err = dopplerRaceBinary(); err != nil {
			t.Skipf("unable to build doppler with the race detector: %v", err)
		}
	}
	p.cmd = exec.Command(binary, args...) // #nosec G204
	p.cmd.Env = env
	p.cmd.Stdout = out
	if opts.ttyStdout {
		if openTTY == nil {
			t.Skip("opening a TTY is not supported on this OS")
		}
		p.cmd.Stdout = openTTY(t)
	}
	p.cmd.Stderr = out
	// run in a new process group so we can signal doppler and its child together, like a TTY does
	p.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := p.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() {
		_ = p.cmd.Wait()
		close(p.done)
	}()

	t.Cleanup(func() {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
		<-p.done
	})
	return p
}

// signal sends a signal to the doppler process only
func (p *dopplerProcess) signal(t *testing.T, sig syscall.Signal) {
	t.Helper()
	if err := p.cmd.Process.Signal(sig); err != nil {
		t.Fatalf("unable to send %v to doppler: %v", sig, err)
	}
}

// signalGroup sends a signal to doppler's whole process group, like pressing Ctrl-C in a terminal does
func (p *dopplerProcess) signalGroup(t *testing.T, sig syscall.Signal) {
	t.Helper()
	if err := syscall.Kill(-p.cmd.Process.Pid, sig); err != nil {
		t.Fatalf("unable to send %v to doppler's process group: %v", sig, err)
	}
}

// spamSignal sends sig every millisecond until doppler exits, to doppler alone or to its whole process group
func (p *dopplerProcess) spamSignal(t *testing.T, sig syscall.Signal, group bool) {
	t.Helper()
	pid := p.cmd.Process.Pid
	if group {
		pid = -pid
	}
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-p.done:
				return
			default:
				_ = syscall.Kill(pid, sig)
				time.Sleep(time.Millisecond)
			}
		}
	}()
}

// mashCtrlC is like a user pressing Ctrl-C repeatedly
func (p *dopplerProcess) mashCtrlC(t *testing.T) {
	t.Helper()
	p.spamSignal(t, syscall.SIGINT, true)
}

func (p *dopplerProcess) events() []string {
	data, err := os.ReadFile(filepath.Join(p.stateDir, "events"))
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func (p *dopplerProcess) hasEvent(event string) bool {
	return slices.Contains(p.events(), event)
}

func (p *dopplerProcess) countEvents(event string) int {
	n := 0
	for _, e := range p.events() {
		if e == event {
			n++
		}
	}
	return n
}

func (p *dopplerProcess) output() string {
	data, _ := os.ReadFile(p.outPath)
	return string(data)
}

func (p *dopplerProcess) describe() string {
	return fmt.Sprintf("child events: %q\ndoppler output:\n%s", p.events(), p.output())
}

// waitForOutput waits until doppler's output contains s at least n times
func (p *dopplerProcess) waitForOutput(t *testing.T, s string, n int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for strings.Count(p.output(), s) < n {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for doppler to log %q %d time(s)\n%s", s, n, p.describe())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (p *dopplerProcess) waitForEvent(t *testing.T, event string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if p.hasEvent(event) {
			return
		}
		select {
		case <-p.done:
			if p.hasEvent(event) {
				return
			}
			t.Fatalf("doppler exited before child logged %q\n%s", event, p.describe())
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("timed out waiting for child to log %q\n%s", event, p.describe())
}

type exitResult struct {
	code     int
	signaled bool
	signal   syscall.Signal
}

func (r exitResult) String() string {
	if r.signaled {
		return fmt.Sprintf("killed by %v", r.signal)
	}
	return fmt.Sprintf("exit code %d", r.code)
}

// waitForExit waits for doppler to exit, failing the test with `why` if it's still running after the timeout
func (p *dopplerProcess) waitForExit(t *testing.T, timeout time.Duration, why string) exitResult {
	t.Helper()
	select {
	case <-p.done:
	case <-time.After(timeout):
		t.Fatalf("%s: doppler was still running after %v\n%s", why, timeout, p.describe())
	}
	status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
	if status.Signaled() {
		return exitResult{signaled: true, signal: status.Signal()}
	}
	return exitResult{code: status.ExitStatus()}
}

// assertRunning fails the test if doppler exits within the given duration
func (p *dopplerProcess) assertRunning(t *testing.T, d time.Duration, why string) {
	t.Helper()
	select {
	case <-p.done:
		status := p.cmd.ProcessState.Sys().(syscall.WaitStatus)
		t.Fatalf("%s: doppler exited unexpectedly (status %v)\n%s", why, status, p.describe())
	case <-time.After(d):
	}
}

func assertExitCode(t *testing.T, p *dopplerProcess, got exitResult, want int) {
	t.Helper()
	if got.signaled || got.code != want {
		t.Fatalf("expected doppler to exit with code %d, got %v\n%s", want, got, p.describe())
	}
}

func assertMountRemoved(t *testing.T, p *dopplerProcess) {
	t.Helper()
	if _, err := os.Lstat(p.mountPath); !os.IsNotExist(err) {
		t.Fatalf("expected secrets mount %s to be removed after doppler exited (lstat err: %v)\n%s", p.mountPath, err, p.describe())
	}
}

// traps that log the signal and exit with a distinct code, killing the background sleep so it doesn't linger
const exitTraps = `trap 'log "TERM $GEN"; kill $! 2>/dev/null; exit 7' TERM
trap 'log "INT $GEN"; kill $! 2>/dev/null; exit 9' INT
trap 'log "HUP $GEN"; kill $! 2>/dev/null; exit 11' HUP
trap 'log "QUIT $GEN"; kill $! 2>/dev/null; exit 13' QUIT
`

// ----------------------------------------------------------------------------------------------------------------
// Current behavior. These pass today and must keep passing.
// ----------------------------------------------------------------------------------------------------------------

func TestRunForwardsTerminationSignalsToChild(t *testing.T) {
	t.Parallel()
	cases := []struct {
		sig      syscall.Signal
		event    string
		exitCode int
	}{
		{syscall.SIGTERM, "TERM 1", 7},
		{syscall.SIGINT, "INT 1", 9},
		{syscall.SIGHUP, "HUP 1", 11},
		{syscall.SIGQUIT, "QUIT 1", 13},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.sig.String(), func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 0)
			p := startDoppler(t, api, runOptions{forwardSignals: true, traps: exitTraps})
			p.waitForEvent(t, "started 1")

			p.signal(t, tc.sig)

			result := p.waitForExit(t, 10*time.Second, "after forwarding "+tc.sig.String())
			assertExitCode(t, p, result, tc.exitCode)
			if !p.hasEvent(tc.event) {
				t.Fatalf("expected child to receive %v\n%s", tc.sig, p.describe())
			}
		})
	}
}

// traps that log non-terminating signals and carry on
const loggingTraps = `trap 'log "USR1 $GEN"' USR1
trap 'log "USR2 $GEN"' USR2
trap 'log "WINCH $GEN"' WINCH
`

// a tail that keeps the child running until the test creates the release file, then exits with code 3.
// unlike `wait`, this isn't cut short by trapped signals
const releaseTail = `log "started $GEN"
while [ ! -f "$STATE/release" ]; do sleep 0.05; done
log "released $GEN"
exit 3
`

func (p *dopplerProcess) release(t *testing.T) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(p.stateDir, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRunForwardsNonTerminatingSignalsToChild(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, traps: exitTraps + loggingTraps, tail: releaseTail})
	p.waitForEvent(t, "started 1")

	for _, tc := range []struct {
		sig   syscall.Signal
		event string
	}{
		{syscall.SIGUSR1, "USR1 1"},
		{syscall.SIGUSR2, "USR2 1"},
		{syscall.SIGWINCH, "WINCH 1"},
	} {
		p.signal(t, tc.sig)
		p.waitForEvent(t, tc.event)
	}
	p.assertRunning(t, 300*time.Millisecond, "SIGUSR1/SIGUSR2/SIGWINCH should not stop doppler")
	for _, e := range []string{"USR1 1", "USR2 1", "WINCH 1"} {
		if n := p.countEvents(e); n != 1 {
			t.Fatalf("expected child to log %q exactly once, got %d\n%s", e, n, p.describe())
		}
	}

	p.release(t)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after releasing child"), 3)
}

func TestRunWithoutForwardingIgnoresSignalsSentOnlyToDoppler(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: false, traps: exitTraps + loggingTraps, tail: releaseTail})
	p.waitForEvent(t, "started 1")

	// includes signals whose default action would terminate doppler (e.g. SIGUSR1)
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGWINCH} {
		p.signal(t, sig)
	}
	p.assertRunning(t, 500*time.Millisecond, "without --forward-signals, signals sent only to doppler are ignored")

	p.release(t)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after releasing child"), 3)
	assertNoSignalsReceived(t, p, 1)
}

func assertNoSignalsReceived(t *testing.T, p *dopplerProcess, gen int) {
	t.Helper()
	for _, name := range []string{"INT", "TERM", "HUP", "QUIT", "USR1", "USR2", "WINCH"} {
		if e := fmt.Sprintf("%s %d", name, gen); p.hasEvent(e) {
			t.Fatalf("child should not have received any signals, but logged %q\n%s", e, p.describe())
		}
	}
}

// assertForwarding sends SIGUSR1 and then SIGTERM to doppler alone, and checks whether they reach the child
func assertForwarding(t *testing.T, opts runOptions, wantForwarded bool) {
	t.Helper()
	api := newMockAPI(t, 0)
	opts.traps = exitTraps + loggingTraps
	opts.tail = releaseTail
	p := startDoppler(t, api, opts)
	p.waitForEvent(t, "started 1")

	p.signal(t, syscall.SIGUSR1)
	if wantForwarded {
		// let the child handle each signal before sending the next, as sh may run pending traps in any order
		p.waitForEvent(t, "USR1 1")
	}
	p.signal(t, syscall.SIGTERM)

	if wantForwarded {
		assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after forwarding SIGTERM"), 7)
		if p.countEvents("USR1 1") != 1 || p.countEvents("TERM 1") != 1 {
			t.Fatalf("expected child to receive SIGUSR1 and SIGTERM exactly once each\n%s", p.describe())
		}
		return
	}

	p.assertRunning(t, 500*time.Millisecond, "signals should not be forwarded")
	p.release(t)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after releasing child"), 3)
	assertNoSignalsReceived(t, p, 1)
}

func TestForwardSignalsFlag(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		opts          runOptions
		wantForwarded bool
	}{
		{"default/stdout-not-tty", runOptions{defaultForwarding: true}, true},
		{"default/stdout-tty", runOptions{defaultForwarding: true, ttyStdout: true}, false},
		{"true/stdout-not-tty", runOptions{forwardSignals: true}, true},
		{"true/stdout-tty", runOptions{forwardSignals: true, ttyStdout: true}, true},
		{"false/stdout-not-tty", runOptions{forwardSignals: false}, false},
		{"false/stdout-tty", runOptions{forwardSignals: false, ttyStdout: true}, false},
	}
	for _, tc := range cases {
		tc := tc
		for _, argsForm := range []bool{false, true} {
			argsForm := argsForm
			name := tc.name + "/command-flag"
			if argsForm {
				name = tc.name + "/args"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				opts := tc.opts
				opts.argsForm = argsForm
				assertForwarding(t, opts, tc.wantForwarded)
			})
		}
	}
}

func TestWatchForwardSignalsFlag(t *testing.T) {
	t.Parallel()
	for _, forward := range []bool{true, false} {
		forward := forward
		t.Run(fmt.Sprintf("forward-signals=%t", forward), func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 0)
			p := startDoppler(t, api, runOptions{forwardSignals: forward, watch: true, traps: exitTraps + loggingTraps, tail: releaseTail})
			p.waitForEvent(t, "started 1")

			// before any restart
			p.signal(t, syscall.SIGUSR1)
			if forward {
				p.waitForEvent(t, "USR1 1")
			}

			// doppler terminates the first child itself when restarting
			triggerSecretsUpdate(t, api, p)
			p.waitForEvent(t, "TERM 1")
			p.waitForEvent(t, "started 2")

			// after a restart, signals go to the new child only when forwarding
			p.signal(t, syscall.SIGUSR1)
			if forward {
				p.waitForEvent(t, "USR1 2")
			}
			p.signal(t, syscall.SIGINT)
			if forward {
				assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after forwarding SIGINT"), 9)
				if p.countEvents("USR1 2") != 1 || p.countEvents("INT 2") != 1 {
					t.Fatalf("expected restarted child to receive SIGUSR1 and SIGINT exactly once each\n%s", p.describe())
				}
				return
			}

			p.assertRunning(t, 500*time.Millisecond, "signals should not be forwarded")
			p.release(t)
			assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after releasing child"), 3)
			assertNoSignalsReceived(t, p, 2)
			if p.hasEvent("USR1 1") {
				t.Fatalf("first child should not have received SIGUSR1\n%s", p.describe())
			}
		})
	}
}

func TestRunWithoutForwardingCtrlCReachesChildOnce(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: false, traps: exitTraps})
	p.waitForEvent(t, "started 1")

	// a TTY sends SIGINT to the whole foreground process group
	p.signalGroup(t, syscall.SIGINT)

	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after Ctrl-C"), 9)
	if n := p.countEvents("INT 1"); n != 1 {
		t.Fatalf("expected child to receive SIGINT exactly once, got %d\n%s", n, p.describe())
	}
}

func TestRunChildKilledBySignalExitsWith255(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	// no traps: the child is killed by the forwarded signal
	p := startDoppler(t, api, runOptions{
		forwardSignals: true,
		tail: `log "started $GEN"
exec sleep 60 >/dev/null 2>&1
`,
	})
	p.waitForEvent(t, "started 1")

	p.signal(t, syscall.SIGTERM)

	// WaitCommand returns -1 for a signaled child, which os.Exit turns into 255
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 255)
}

func TestRunRepeatedCtrlC(t *testing.T) {
	t.Parallel()
	for _, watch := range []bool{false, true} {
		for _, forward := range []bool{false, true} {
			watch, forward := watch, forward
			// the window between the child exiting and doppler exiting is tiny, so run several attempts to make a
			// regression here likely to be caught
			for i := 0; i < 5; i++ {
				t.Run(fmt.Sprintf("watch=%t/forward-signals=%t/attempt-%d", watch, forward, i), func(t *testing.T) {
					t.Parallel()
					api := newMockAPI(t, 0)
					p := startDoppler(t, api, runOptions{
						forwardSignals: forward,
						watch:          watch,
						// the child takes a moment to shut down after the first Ctrl-C and ignores the rest, which
						// keeps the Ctrl-Cs coming while it exits
						traps: `trap 'trap "" INT; log "INT $GEN"; kill $! 2>/dev/null; sleep 0.2; exit 9' INT`,
					})
					p.waitForEvent(t, "started 1")

					p.mashCtrlC(t)

					// doppler exits with the child's exit code; no Ctrl-C may terminate doppler itself
					assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after repeated Ctrl-C"), 9)
					if p.hasEvent("started 2") {
						t.Fatalf("doppler should not have started another child\n%s", p.describe())
					}
				})
			}
		}
	}
}

func TestRunRepeatedCtrlCBeforeChildStarts(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 1)
	p := startDoppler(t, api, runOptions{traps: exitTraps})
	api.waitForDownload(t, 1)

	p.mashCtrlC(t)

	result := p.waitForExit(t, 5*time.Second, "after repeated Ctrl-C while fetching secrets")
	if !result.signaled || result.signal != syscall.SIGINT {
		t.Fatalf("expected doppler to be terminated by SIGINT, got %v\n%s", result, p.describe())
	}
	if p.hasEvent("started 1") {
		t.Fatalf("child should never have started\n%s", p.describe())
	}
}

func TestRunExitCodeUnaffectedBySignalsWhileChildExits(t *testing.T) {
	t.Parallel()
	// the window between the child exiting and doppler exiting is tiny, so run several attempts to make a
	// regression here likely to be caught
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("attempt-%d", i), func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 0)
			p := startDoppler(t, api, runOptions{
				forwardSignals: true,
				// ignore further SIGTERMs once shutdown begins, otherwise each one would restart the trap
				traps: `trap 'trap "" TERM; log "TERM $GEN"; kill $! 2>/dev/null; sleep 0.2; exit 7' TERM`,
			})
			p.waitForEvent(t, "started 1")

			// keep signaling doppler until it exits. none of these may terminate doppler itself, including any that land
			// after the child has exited but before doppler does
			p.spamSignal(t, syscall.SIGTERM, false)

			assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after repeated SIGTERM"), 7)
		})
	}
}

func TestRunTerminationSignalBeforeChildStarts(t *testing.T) {
	t.Parallel()
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		sig := sig
		t.Run(sig.String(), func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 1)
			p := startDoppler(t, api, runOptions{forwardSignals: true, traps: exitTraps})
			api.waitForDownload(t, 1)

			p.signal(t, sig)

			result := p.waitForExit(t, 5*time.Second, "after "+sig.String()+" while fetching secrets")
			if !result.signaled || result.signal != sig {
				t.Fatalf("expected doppler to be terminated by %v, got %v\n%s", sig, result, p.describe())
			}
			if p.hasEvent("started 1") {
				t.Fatalf("child should never have started\n%s", p.describe())
			}
		})
	}
}

func TestRunNonTerminatingSignalBeforeChildStarts(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 1)
	p := startDoppler(t, api, runOptions{forwardSignals: true, traps: exitTraps})
	api.waitForDownload(t, 1)

	// these signals are ignored by default and must not stop doppler
	p.signal(t, syscall.SIGWINCH)
	p.signal(t, syscall.SIGCHLD)
	p.signal(t, syscall.SIGURG)
	p.assertRunning(t, 300*time.Millisecond, "SIGWINCH/SIGCHLD/SIGURG should not stop doppler")

	api.releaseDownloads()
	p.waitForEvent(t, "started 1")
	p.signal(t, syscall.SIGTERM)
	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 7)
}

func TestRunMountIsRemovedAfterForwardedSignal(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, mount: true, traps: exitTraps})
	p.waitForEvent(t, "started 1") // GEN is read from the mount, so this also proves the mount was readable

	if fi, err := os.Lstat(p.mountPath); err != nil || fi.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("expected secrets mount to be a named pipe while the child runs (err: %v)", err)
	}

	p.signal(t, syscall.SIGTERM)

	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 7)
	assertMountRemoved(t, p)
}

func TestWatchForwardsSignalsToRestartedChild(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, watch: true, traps: exitTraps})
	p.waitForEvent(t, "started 1")

	triggerSecretsUpdate(t, api, p)
	p.waitForEvent(t, "TERM 1") // doppler terminates the old child itself
	p.waitForEvent(t, "started 2")

	p.signal(t, syscall.SIGINT)

	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGINT"), 9)
	if !p.hasEvent("INT 2") {
		t.Fatalf("expected the restarted child to receive SIGINT\n%s", p.describe())
	}
	if p.hasEvent("INT 1") {
		t.Fatalf("the old child should not have received SIGINT\n%s", p.describe())
	}
}

func TestWatchWithoutForwardingCtrlCReachesRestartedChildOnce(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: false, watch: true, traps: exitTraps})
	p.waitForEvent(t, "started 1")

	triggerSecretsUpdate(t, api, p)
	p.waitForEvent(t, "started 2")

	p.signalGroup(t, syscall.SIGINT)

	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after Ctrl-C"), 9)
	if n := p.countEvents("INT 2"); n != 1 {
		t.Fatalf("expected the restarted child to receive SIGINT exactly once, got %d\n%s", n, p.describe())
	}
}

func TestWatchRestartRemountsSecrets(t *testing.T) {
	t.Parallel()
	api := newMockAPI(t, 0)
	p := startDoppler(t, api, runOptions{forwardSignals: true, watch: true, mount: true, traps: exitTraps})
	p.waitForEvent(t, "started 1")

	triggerSecretsUpdate(t, api, p)
	p.waitForEvent(t, "started 2") // GEN=2 was read from the new mount

	p.signal(t, syscall.SIGTERM)

	assertExitCode(t, p, p.waitForExit(t, 10*time.Second, "after SIGTERM"), 7)
	assertMountRemoved(t, p)
}

// ----------------------------------------------------------------------------------------------------------------
// Known bugs. These fail today: a termination signal that arrives while --watch is restarting the child is
// swallowed, and doppler carries on and starts a new child.
// ----------------------------------------------------------------------------------------------------------------

// the old child takes a while to shut down when doppler terminates it, which holds the restart open
const slowShutdownTraps = `trap 'log "TERMSTART $GEN"; kill $! 2>/dev/null; sleep 1; log "TERMEND $GEN"; exit 7' TERM
trap 'log "INT $GEN"; kill $! 2>/dev/null; exit 9' INT
`

func TestWatchSignalDuringRestartStopsDoppler(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		forwardSignals bool
		mount          bool
		send           func(t *testing.T, p *dopplerProcess)
		// with no child to defer to, doppler is terminated by the signal it received
		want exitResult
	}{
		{"forwarding/SIGINT", true, false, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGINT) }, killedBy(syscall.SIGINT)},
		{"forwarding/SIGTERM", true, false, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGTERM) }, killedBy(syscall.SIGTERM)},
		{"forwarding/SIGHUP", true, false, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGHUP) }, killedBy(syscall.SIGHUP)},
		// the Go runtime would dump goroutines on an unhandled SIGQUIT, so doppler exits with 128+3 instead
		{"forwarding/SIGQUIT", true, false, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGQUIT) }, exitResult{code: 131}},
		{"no-forwarding/Ctrl-C", false, false, func(t *testing.T, p *dopplerProcess) { p.signalGroup(t, syscall.SIGINT) }, killedBy(syscall.SIGINT)},
		{"forwarding/SIGINT/mount", true, true, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGINT) }, killedBy(syscall.SIGINT)},
		{"no-forwarding/Ctrl-C/mount", false, true, func(t *testing.T, p *dopplerProcess) { p.signalGroup(t, syscall.SIGINT) }, killedBy(syscall.SIGINT)},
		{"no-forwarding/repeated-Ctrl-C/mount", false, true, func(t *testing.T, p *dopplerProcess) { p.mashCtrlC(t) }, killedBy(syscall.SIGINT)},
		{"forwarding/repeated-Ctrl-C/mount", true, true, func(t *testing.T, p *dopplerProcess) { p.mashCtrlC(t) }, killedBy(syscall.SIGINT)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 0)
			p := startDoppler(t, api, runOptions{forwardSignals: tc.forwardSignals, watch: true, mount: tc.mount, traps: slowShutdownTraps})
			p.waitForEvent(t, "started 1")

			triggerSecretsUpdate(t, api, p)
			p.waitForEvent(t, "TERMSTART 1")

			tc.send(t, p)

			result := p.waitForExit(t, 8*time.Second, "termination signal during a --watch restart should stop doppler")
			assertStoppedDuringRestart(t, p, result, tc.want, tc.mount)
		})
	}
}

func killedBy(sig syscall.Signal) exitResult {
	return exitResult{signaled: true, signal: sig}
}

func assertStoppedDuringRestart(t *testing.T, p *dopplerProcess, got exitResult, want exitResult, mount bool) {
	t.Helper()
	if got != want {
		t.Fatalf("expected doppler to exit with %v, got %v\n%s", want, got, p.describe())
	}
	if strings.Contains(p.output(), "SIGQUIT: quit") {
		t.Fatalf("doppler should exit cleanly, without a goroutine dump\n%s", p.describe())
	}
	if p.hasEvent("started 2") {
		t.Fatalf("doppler should not start a new child after being told to stop\n%s", p.describe())
	}
	if mount {
		assertMountRemoved(t, p)
	}
}

// A second secrets.update that arrives while a restart is in progress queues another restart, which fetches secrets
// before starting a child. Doppler must not wait for that fetch once it's been told to stop. The mock holds the
// queued update's download (the 3rd), so waiting for it would hang.
func TestWatchSignalDuringRestartWithQueuedUpdate(t *testing.T) {
	t.Parallel()
	for _, mount := range []bool{false, true} {
		mount := mount
		t.Run(fmt.Sprintf("mount=%t", mount), func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 3)
			p := startDoppler(t, api, runOptions{forwardSignals: true, watch: true, mount: mount, traps: slowShutdownTraps})
			p.waitForEvent(t, "started 1")

			triggerSecretsUpdate(t, api, p)
			p.waitForEvent(t, "TERMSTART 1")

			// asked to stop while the old child is still shutting down, then another update arrives
			p.signal(t, syscall.SIGINT)
			triggerSecretsUpdate(t, api, p)

			result := p.waitForExit(t, 5*time.Second, "doppler should exit once the old child is gone")
			assertStoppedDuringRestart(t, p, result, killedBy(syscall.SIGINT), mount)
		})
	}
}

func TestWatchSignalWhileFetchingQueuedUpdate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		forwardSignals bool
		mount          bool
		send           func(t *testing.T, p *dopplerProcess)
	}{
		{"forwarding/SIGINT", true, false, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGINT) }},
		{"no-forwarding/Ctrl-C", false, false, func(t *testing.T, p *dopplerProcess) { p.signalGroup(t, syscall.SIGINT) }},
		{"forwarding/SIGINT/mount", true, true, func(t *testing.T, p *dopplerProcess) { p.signal(t, syscall.SIGINT) }},
		{"no-forwarding/Ctrl-C/mount", false, true, func(t *testing.T, p *dopplerProcess) { p.signalGroup(t, syscall.SIGINT) }},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			api := newMockAPI(t, 3)
			p := startDoppler(t, api, runOptions{forwardSignals: tc.forwardSignals, watch: true, mount: tc.mount, traps: slowShutdownTraps})
			p.waitForEvent(t, "started 1")

			// queue a second update while the first restart waits for the old child to shut down
			triggerSecretsUpdate(t, api, p)
			p.waitForEvent(t, "TERMSTART 1")
			triggerSecretsUpdate(t, api, p)

			// the old child is gone and the queued update is fetching secrets, so no child is running
			p.waitForEvent(t, "TERMEND 1")
			api.waitForDownload(t, 3)

			tc.send(t, p)

			result := p.waitForExit(t, 3*time.Second, "doppler should exit while fetching secrets for a queued restart")
			assertStoppedDuringRestart(t, p, result, killedBy(syscall.SIGINT), tc.mount)
		})
	}
}
