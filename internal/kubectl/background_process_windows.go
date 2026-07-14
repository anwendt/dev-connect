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
