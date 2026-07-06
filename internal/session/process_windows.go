//go:build windows

package session

// ProcessExists reports whether pid may refer to a running process.
//
// The first implementation is conservative on Windows to avoid deleting session
// state for a process that cannot be inspected through the initial abstraction.
func ProcessExists(pid int) bool {
	return pid > 0
}
