package port

import (
	"fmt"
	"net"
)

// Allocation describes a loopback TCP port allocation.
type Allocation struct {
	Host string
	Port int
}

// TCPAddr returns the allocation as a TCP address.
func (allocation Allocation) TCPAddr() *net.TCPAddr {
	return &net.TCPAddr{IP: net.ParseIP(allocation.Host), Port: allocation.Port}
}

// AllocateLoopback allocates a free local loopback TCP port.
func AllocateLoopback() (Allocation, error) {
	return allocateLoopback(defaultPortProbe)
}

func allocateLoopback(probe func() (int, error)) (Allocation, error) {
	port, err := probe()
	if err != nil {
		return Allocation{}, err
	}
	return Allocation{Host: "127.0.0.1", Port: port}, nil
}

func defaultPortProbe() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("resolve loopback: %w", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("listen on loopback: %w", err)
	}
	defer func() {
		_ = listener.Close()
	}()

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}

	return tcpAddr.Port, nil
}
