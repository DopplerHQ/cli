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
package cmd_test

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"unsafe"
)

func init() {
	openTTY = func(t *testing.T) *os.File {
		t.Helper()
		ptmx, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			t.Skipf("unable to open a pseudo-terminal: %v", err)
		}
		t.Cleanup(func() { ptmx.Close() })

		var unlock int32
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, ptmx.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); errno != 0 {
			t.Fatalf("unable to unlock pseudo-terminal: %v", errno)
		}
		var n uint32
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, ptmx.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); errno != 0 {
			t.Fatalf("unable to get pseudo-terminal number: %v", errno)
		}

		tty, err := os.OpenFile(fmt.Sprintf("/dev/pts/%d", n), os.O_RDWR|syscall.O_NOCTTY, 0)
		if err != nil {
			t.Fatalf("unable to open pseudo-terminal: %v", err)
		}
		t.Cleanup(func() { tty.Close() })
		return tty
	}
}
