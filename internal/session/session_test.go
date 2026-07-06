package session

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestStoreRoundTripJSONState(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	state := State{
		SessionID:         "session-1",
		Target:            "dev01",
		Gateway:           "dev-connect-gateway-dev01",
		Namespace:         "dev-connect",
		KubernetesContext: "platform-dev",
		LocalPort:         55221,
		SSHConfigPath:     "/tmp/dev-connect/ssh_config",
		KnownHostsPath:    "/tmp/dev-connect/known_hosts",
		StartedAt:         time.Unix(100, 0).UTC(),
		Reconnect:         true,
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if got.SessionID != state.SessionID || got.Target != state.Target || got.LocalPort != state.LocalPort {
		t.Fatalf("state mismatch: got %#v want %#v", got, state)
	}
	if got.SSHConfigPath != state.SSHConfigPath || got.KnownHostsPath != state.KnownHostsPath {
		t.Fatalf("SSH file paths mismatch: got %#v want %#v", got, state)
	}
}

func TestStateRejectsCredentialFields(t *testing.T) {
	tests := []struct {
		name  string
		state State
		want  string
	}{
		{
			name:  "private key",
			state: State{SessionID: "session-1", Target: "dev01", SSHPrivateKey: "forbidden"},
			want:  "SSH private keys",
		},
		{
			name:  "password",
			state: State{SessionID: "session-1", Target: "dev01", Password: "forbidden"},
			want:  "passwords",
		},
		{
			name:  "kubernetes token",
			state: State{SessionID: "session-1", Target: "dev01", KubernetesToken: "forbidden"},
			want:  "Kubernetes tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if err == nil {
				t.Fatal("state with credential field accepted")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestStoreSaveRejectsCredentialFields(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	err := store.Save(State{SessionID: "session-1", Target: "dev01", Password: "forbidden"})
	if err == nil {
		t.Fatal("Save accepted credential-bearing state")
	}
	if _, statErr := os.Stat(filepath.Join(store.Dir, stateFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("state file exists after rejected save: %v", statErr)
	}
}

func TestStoreClearRemovesState(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	if err := store.Save(State{SessionID: "session-1", Target: "dev01"}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("clear state: %v", err)
	}
	if _, err := store.Load(); !errors.Is(err, ErrNotFound) {
		t.Fatalf("load after clear error = %v, want ErrNotFound", err)
	}
}

func TestStoreClearIgnoresMissingState(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	if err := store.Clear(); err != nil {
		t.Fatalf("clear missing state: %v", err)
	}
}

func TestStoreLoadRejectsInvalidJSON(t *testing.T) {
	store := Store{Dir: t.TempDir()}
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(store.Dir, stateFileName), []byte("{"), 0o600); err != nil {
		t.Fatalf("write invalid JSON: %v", err)
	}
	_, err := store.Load()
	if err == nil {
		t.Fatal("Load accepted invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse session state") {
		t.Fatalf("error = %q, want parse session state", err)
	}
}

func TestStoreSaveCreatesRestrictedStateFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not reliable on Windows")
	}

	store := Store{Dir: t.TempDir()}
	if err := store.Save(State{SessionID: "session-1", Target: "dev01"}); err != nil {
		t.Fatalf("save state: %v", err)
	}

	info, err := os.Stat(filepath.Join(store.Dir, stateFileName))
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if got := info.Mode().Perm(); got != stateFileMode {
		t.Fatalf("file mode = %#o, want %#o", got, stateFileMode)
	}
}

func TestLockAllowsOneActiveSession(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "session.lock")

	first, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	defer func() {
		_ = first.Release()
	}()

	if _, err := AcquireLock(lockPath); !errors.Is(err, ErrLocked) {
		t.Fatalf("second acquire error = %v, want ErrLocked", err)
	}

	if err := first.Release(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}
	second, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("release second lock: %v", err)
	}
}

func TestStaleStateDetection(t *testing.T) {
	state := State{PortForwardPID: 42}
	if !state.IsStale(func(pid int) bool { return pid != 42 }) {
		t.Fatal("state with missing PID is not stale")
	}
	if state.IsStale(func(pid int) bool { return pid == 42 }) {
		t.Fatal("state with running PID is stale")
	}
}

func TestStateWithoutProcessIsNotStale(t *testing.T) {
	state := State{}
	if state.IsStale(func(int) bool {
		t.Fatal("processExists should not be called for empty PID")
		return false
	}) {
		t.Fatal("state without PID is stale")
	}
}

func TestTerminateProcessIgnoresEmptyPID(t *testing.T) {
	if err := TerminateProcess(0); err != nil {
		t.Fatalf("terminate empty PID: %v", err)
	}
}

func TestReleaseNilLockIsNoop(t *testing.T) {
	var lock *Lock
	if err := lock.Release(); err != nil {
		t.Fatalf("release nil lock: %v", err)
	}
}

func TestReleaseAlreadyReleasedLockIsNoop(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "session.lock")
	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquire lock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release already released lock: %v", err)
	}
}
