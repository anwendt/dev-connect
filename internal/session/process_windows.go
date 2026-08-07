// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>

//go:build windows

package session

import (
	"os/exec"
	"strconv"

	"golang.org/x/sys/windows"
)

// ProcessExists reports whether pid currently refers to a running process.
func ProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = windows.CloseHandle(handle)
	return true
}

func terminateProcess(pid int) error {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	return cmd.Run()
}
