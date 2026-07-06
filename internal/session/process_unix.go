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
