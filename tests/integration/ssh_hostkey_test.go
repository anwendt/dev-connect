package integration_test

import (
	"net"
	"testing"
	"time"

	"github.com/anwendt/dev-connect/internal/testsupport/sshserver"
	"golang.org/x/crypto/ssh"
)

func TestMockSSHServerAcceptsPinnedHostKey(t *testing.T) {
	signer := newSigner(t)
	server := startSSHServer(t, signer)

	if err := dialWithPinnedHostKey(server.Addr(), signer.PublicKey()); err != nil {
		t.Fatalf("dial with pinned host key: %v", err)
	}
}

func TestMockSSHServerRejectsUnknownHostKey(t *testing.T) {
	serverSigner := newSigner(t)
	pinnedSigner := newSigner(t)
	server := startSSHServer(t, serverSigner)

	if err := dialWithPinnedHostKey(server.Addr(), pinnedSigner.PublicKey()); err == nil {
		t.Fatal("dial with unknown host key succeeded")
	}
}

func TestMockSSHServerSupportsHostKeyRotation(t *testing.T) {
	oldSigner := newSigner(t)
	newSigner := newSigner(t)
	server := startSSHServer(t, newSigner)

	if err := dialWithPinnedHostKey(server.Addr(), oldSigner.PublicKey()); err == nil {
		t.Fatal("dial with obsolete host key succeeded")
	}
	if err := dialWithPinnedHostKey(server.Addr(), newSigner.PublicKey()); err != nil {
		t.Fatalf("dial with rotated host key: %v", err)
	}
}

func newSigner(t *testing.T) ssh.Signer {
	t.Helper()

	signer, err := sshserver.NewSigner()
	if err != nil {
		t.Fatalf("create signer: %v", err)
	}
	return signer
}

func startSSHServer(t *testing.T, signer ssh.Signer) *sshserver.Server {
	t.Helper()

	server, err := sshserver.Start(signer)
	if err != nil {
		t.Fatalf("start SSH server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Fatalf("close SSH server: %v", err)
		}
	})
	return server
}

func dialWithPinnedHostKey(addr string, pinnedKey ssh.PublicKey) error {
	config := &ssh.ClientConfig{
		User:            "developer",
		Auth:            nil,
		HostKeyCallback: ssh.FixedHostKey(pinnedKey),
		Timeout:         5 * time.Second,
	}
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return err
	}
	return conn.Close()
}

func TestMockSSHServerListensOnLoopback(t *testing.T) {
	server := startSSHServer(t, newSigner(t))
	host, _, err := net.SplitHostPort(server.Addr())
	if err != nil {
		t.Fatalf("split server address: %v", err)
	}
	if host != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", host)
	}
}
