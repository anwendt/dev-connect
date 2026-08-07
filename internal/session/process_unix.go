// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>

//go:build !windows

package session

import (
	"errors"
	"syscall"
)

// ProcessExists reports whether pid currently refers to a running process.
func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return !errors.Is(err, syscall.ESRCH)
}

func terminateProcess(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return syscall.Kill(pid, syscall.SIGKILL)
}
