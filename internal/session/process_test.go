// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>

package session

import (
	"os"
	"testing"
)

func TestProcessExistsReportsCurrentProcess(t *testing.T) {
	if !ProcessExists(os.Getpid()) {
		t.Fatal("current process was not detected")
	}
}

func TestProcessExistsRejectsEmptyPID(t *testing.T) {
	if ProcessExists(0) {
		t.Fatal("empty PID was detected as running")
	}
}
