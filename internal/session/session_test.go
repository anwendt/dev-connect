package session

import (
	"errors"
	"path/filepath"
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
}

func TestStateRejectsCredentialFields(t *testing.T) {
	state := State{SessionID: "session-1", Target: "dev01", SSHPrivateKey: "forbidden"}

	if err := state.Validate(); err == nil {
		t.Fatal("state with private key accepted")
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
	if _, err := AcquireLock(lockPath); err != nil {
		t.Fatalf("acquire after release: %v", err)
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
