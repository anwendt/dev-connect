package sshserver

import (
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestNewSignerCreatesUsableSigner(t *testing.T) {
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	if signer.PublicKey() == nil {
		t.Fatal("signer public key is nil")
	}
	if signer.PublicKey().Type() != ssh.KeyAlgoED25519 {
		t.Fatalf("signer type = %q, want %q", signer.PublicKey().Type(), ssh.KeyAlgoED25519)
	}
}

func TestStartRejectsNilSigner(t *testing.T) {
	if _, err := Start(nil); err == nil {
		t.Fatal("Start accepted nil signer")
	}
}

func TestServerAcceptsSSHSessionAndWritesBanner(t *testing.T) {
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	server, err := Start(signer)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	client, err := ssh.Dial("tcp", server.Addr(), &ssh.ClientConfig{
		User:            "developer",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("ssh dial: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	channel, requests, err := client.OpenChannel("session", nil)
	if err != nil {
		t.Fatalf("open session channel: %v", err)
	}
	defer func() {
		_ = channel.Close()
	}()
	go ssh.DiscardRequests(requests)

	output, err := io.ReadAll(channel)
	if err != nil {
		t.Fatalf("read channel: %v", err)
	}
	if !strings.Contains(string(output), "dev-connect mock ssh server") {
		t.Fatalf("output = %q, want mock banner", string(output))
	}
}

func TestServerRejectsUnsupportedChannel(t *testing.T) {
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	server, err := Start(signer)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	client, err := ssh.Dial("tcp", server.Addr(), &ssh.ClientConfig{
		User:            "developer",
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatalf("ssh dial: %v", err)
	}
	defer func() {
		_ = client.Close()
	}()

	channel, requests, err := client.OpenChannel("unsupported", nil)
	if err == nil {
		_ = channel.Close()
		go ssh.DiscardRequests(requests)
		t.Fatal("unsupported channel was accepted")
	}
	if !strings.Contains(err.Error(), "unsupported channel") {
		t.Fatalf("error = %q, want unsupported channel", err)
	}
}

func TestServerIgnoresInvalidSSHHandshake(t *testing.T) {
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	server, err := Start(signer)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		_ = server.Close()
	}()

	conn, err := net.DialTimeout("tcp", server.Addr(), 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := conn.Write([]byte("not ssh\n")); err != nil {
		t.Fatalf("write invalid handshake: %v", err)
	}
	_ = conn.Close()
}

func TestServerCloseIsIdempotent(t *testing.T) {
	signer, err := NewSigner()
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	server, err := Start(signer)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if server.Addr() == "" {
		t.Fatal("server address is empty")
	}
	if err := server.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
