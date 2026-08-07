// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>

//go:build windows

package kubectl

import (
	"os/exec"
	"syscall"
)

const createNewProcessGroup = 0x00000200

func configureBackgroundProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
}
