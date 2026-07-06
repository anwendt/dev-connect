// Package sshserver provides a lightweight SSH server for integration tests.
package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Server is a deterministic mock SSH server for tests.
type Server struct {
	listener net.Listener
	config   *ssh.ServerConfig
	done     chan struct{}
	once     sync.Once
}

// NewSigner creates an ephemeral Ed25519 host key signer.
func NewSigner() (ssh.Signer, error) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate host key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("create SSH signer: %w", err)
	}
	return signer, nil
}

// Start starts a mock SSH server on the loopback interface.
func Start(signer ssh.Signer) (*Server, error) {
	if signer == nil {
		return nil, fmt.Errorf("host key signer is required")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	server := &Server{
		listener: listener,
		config: &ssh.ServerConfig{
			NoClientAuth:  true,
			ServerVersion: "SSH-2.0-dev-connect-test",
		},
		done: make(chan struct{}),
	}
	server.config.AddHostKey(signer)
	go server.serve()
	return server, nil
}

// Addr returns the listener address.
func (server *Server) Addr() string {
	return server.listener.Addr().String()
}

// Close stops the server.
func (server *Server) Close() error {
	var err error
	server.once.Do(func() {
		err = server.listener.Close()
		<-server.done
	})
	return err
}

func (server *Server) serve() {
	defer close(server.done)
	for {
		conn, err := server.listener.Accept()
		if err != nil {
			return
		}
		go server.handle(conn)
	}
}

func (server *Server) handle(conn net.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	sshConn, channels, requests, err := ssh.NewServerConn(conn, server.config)
	if err != nil {
		return
	}
	defer func() {
		_ = sshConn.Close()
	}()
	go ssh.DiscardRequests(requests)
	for channel := range channels {
		if channel.ChannelType() != "session" {
			_ = channel.Reject(ssh.UnknownChannelType, "unsupported channel")
			continue
		}
		sessionChannel, requests, err := channel.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(requests)
		_, _ = io.WriteString(sessionChannel, "dev-connect mock ssh server\n")
		_ = sessionChannel.Close()
	}
}
