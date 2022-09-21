//go:build !windows
// +build !windows

/*
Copyright © 2021 Doppler <support@doppler.com>

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
	"errors"
	"os"
	"syscall"
)

const SupportsNamedPipes = true

func FileOwnership(path string) (int, int, error) {
	info, err := os.Stat(path)
	if err != nil {
		return -1, -1, err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return -1, -1, errors.New("Unable to stat file")
	}

	return int(stat.Uid), int(stat.Gid), nil
}

func CreateNamedPipe(path string, mode uint32) error {
	// this path must be cleaned up manually, but the pipe's contents are
	// only available while the writer (i.e. this program) is alive
	return syscall.Mkfifo(path, mode)
}
