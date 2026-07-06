package port

import "testing"

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
