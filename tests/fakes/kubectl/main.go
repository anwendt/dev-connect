// Package main provides a Go-based fake kubectl executable for tests.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	args := os.Args[1:]
	recordInvocation(args)

	joined := strings.Join(args, " ")
	switch {
	case contains(args, "version"):
		fmt.Println("Client Version: v1.30.0")
		fmt.Println("Server Version: v1.30.0")
	case strings.Contains(joined, "auth can-i create pods/portforward"):
		canI := os.Getenv("DEV_CONNECT_FAKE_KUBECTL_CAN_I")
		if canI == "" {
			canI = "yes"
		}
		fmt.Println(canI)
		if canI != "yes" {
			os.Exit(1)
		}
	case contains(args, "config") && contains(args, "current-context"):
		contextName := os.Getenv("DEV_CONNECT_FAKE_KUBECTL_CONTEXT")
		if contextName == "" {
			contextName = "rancher-dev"
		}
		fmt.Println(contextName)
	case contains(args, "port-forward"):
		localPort := "55221"
		for _, arg := range args {
			if strings.Contains(arg, ":") {
				localPort = strings.SplitN(arg, ":", 2)[0]
			}
		}
		fmt.Printf("Forwarding from 127.0.0.1:%s -> 22\n", localPort)
		blockUntilStopped()
	default:
		fmt.Fprintf(os.Stderr, "unexpected kubectl args: %s\n", joined)
		os.Exit(9)
	}
}

func contains(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func recordInvocation(args []string) {
	path := os.Getenv("DEV_CONNECT_FAKE_KUBECTL_LOG")
	if path == "" {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()
	_, _ = fmt.Fprintln(file, strings.Join(args, "\x00"))
}

func blockUntilStopped() {
	for {
		time.Sleep(time.Hour)
	}
}
