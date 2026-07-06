package port

import (
	"errors"
	"net"
	"testing"
)

func TestAllocateLoopbackReturnsUsableNonPrivilegedPort(t *testing.T) {
	allocation, err := allocateLoopback(func() (int, error) {
		return 55221, nil
	})
	if err != nil {
		t.Fatalf("allocate loopback: %v", err)
	}

	if allocation.Host != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", allocation.Host)
	}
	if allocation.Port <= 1024 {
		t.Fatalf("port = %d, want non-privileged port", allocation.Port)
	}
	if allocation.TCPAddr().String() != "127.0.0.1:55221" {
		t.Fatalf("TCPAddr = %s, want 127.0.0.1:55221", allocation.TCPAddr())
	}
}

func TestAllocateLoopbackPropagatesProbeError(t *testing.T) {
	wantErr := errors.New("probe failed")
	_, err := allocateLoopback(func() (int, error) {
		return 0, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("allocateLoopback error = %v, want %v", err, wantErr)
	}
}

func TestAllocateLoopbackReturnsBindablePort(t *testing.T) {
	allocation, err := AllocateLoopback()
	if err != nil {
		t.Fatalf("AllocateLoopback: %v", err)
	}
	if allocation.Host != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", allocation.Host)
	}
	if allocation.Port <= 0 {
		t.Fatalf("port = %d, want positive port", allocation.Port)
	}

	listener, err := net.ListenTCP("tcp", allocation.TCPAddr())
	if err != nil {
		t.Fatalf("allocated port was not bindable: %v", err)
	}
	_ = listener.Close()
}

func TestTCPAddrHandlesInvalidHost(t *testing.T) {
	addr := Allocation{Host: "not-an-ip", Port: 22}.TCPAddr()
	if addr.IP != nil {
		t.Fatalf("IP = %v, want nil for invalid host", addr.IP)
	}
	if addr.Port != 22 {
		t.Fatalf("Port = %d, want 22", addr.Port)
	}
}
