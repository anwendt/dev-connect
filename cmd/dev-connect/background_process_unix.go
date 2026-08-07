// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>

//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func configureDetachedCommand(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
