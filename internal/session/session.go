package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	stateFileName = "session.json"
	stateFileMode = 0o600
)

// ErrNotFound indicates no session state exists.
var ErrNotFound = errors.New("session state not found")

// ErrLocked indicates another managed session is active.
var ErrLocked = errors.New("session lock already held")

// State describes local managed session metadata.
type State struct {
	SessionID         string    `json:"sessionId"`
	Target            string    `json:"target"`
	Gateway           string    `json:"gateway,omitempty"`
	Namespace         string    `json:"namespace,omitempty"`
	KubernetesContext string    `json:"kubernetesContext,omitempty"`
	LocalPort         int       `json:"localPort,omitempty"`
	PortForwardPID    int       `json:"portForwardPid,omitempty"`
	SSHConfigPath     string    `json:"sshConfigPath,omitempty"`
	KnownHostsPath    string    `json:"knownHostsPath,omitempty"`
	UserSSHConfigPath string    `json:"userSshConfigPath,omitempty"`
	VSCodeUserDataDir string    `json:"vscodeUserDataDir,omitempty"`
	StartedAt         time.Time `json:"startedAt,omitempty"`
	Reconnect         bool      `json:"reconnect"`
	SSHPrivateKey     string    `json:"-"`
	Password          string    `json:"-"`
	KubernetesToken   string    `json:"-"`
}

// Validate rejects credential-bearing session state.
func (state State) Validate() error {
	if state.SSHPrivateKey != "" {
		return errors.New("session state must not contain SSH private keys")
	}
	if state.Password != "" {
		return errors.New("session state must not contain passwords")
	}
	if state.KubernetesToken != "" {
		return errors.New("session state must not contain Kubernetes tokens")
	}
	return nil
}

// IsStale reports whether a session state references a missing process.
func (state State) IsStale(processExists func(pid int) bool) bool {
	if state.PortForwardPID <= 0 {
		return false
	}
	return !processExists(state.PortForwardPID)
}

// TerminateProcess attempts to stop a recorded session process.
func TerminateProcess(pid int) error {
	if pid <= 0 {
		return nil
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("terminate process %d: %w", pid, err)
	}
	return nil
}

// Store manages local JSON session state.
type Store struct {
	Dir string
}

// Save writes session state as JSON.
func (store Store) Save(state State) error {
	if err := state.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		return fmt.Errorf("create session directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session state: %w", err)
	}
	if err := os.WriteFile(store.path(), data, stateFileMode); err != nil {
		return fmt.Errorf("write session state: %w", err)
	}
	return os.Chmod(store.path(), stateFileMode)
}

// Load reads session state JSON.
func (store Store) Load() (State, error) {
	data, err := os.ReadFile(store.path())
	if errors.Is(err, os.ErrNotExist) {
		return State{}, ErrNotFound
	}
	if err != nil {
		return State{}, fmt.Errorf("read session state: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse session state: %w", err)
	}
	return state, nil
}

// Clear removes session state.
func (store Store) Clear() error {
	if err := os.Remove(store.path()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session state: %w", err)
	}
	return nil
}

func (store Store) path() string {
	return filepath.Join(store.Dir, stateFileName)
}

// Lock represents an acquired session lock.
type Lock struct {
	path string
	file *os.File
}

// AcquireLock creates an exclusive session lock file.
func AcquireLock(path string) (*Lock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil, ErrLocked
	}
	if err != nil {
		return nil, fmt.Errorf("acquire session lock: %w", err)
	}
	return &Lock{path: path, file: file}, nil
}

// Release releases the session lock.
func (lock *Lock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	if err := lock.file.Close(); err != nil {
		return fmt.Errorf("close session lock: %w", err)
	}
	lock.file = nil
	if err := os.Remove(lock.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove session lock: %w", err)
	}
	return nil
}
