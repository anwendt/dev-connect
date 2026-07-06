package kubectl

import (
	"context"
	"errors"
	"testing"
)

func TestBuildPortForwardServiceCommand(t *testing.T) {
	cmd := PortForwardServiceCommand(PortForwardOptions{
		Namespace:   "dev-connect",
		Service:     "dev-connect-gateway-dev01",
		LocalPort:   55221,
		RemotePort:  22,
		ContextName: "platform-dev",
		Kubeconfig:  "/tmp/kubeconfig",
	})

	want := []string{
		"--kubeconfig", "/tmp/kubeconfig",
		"--context", "platform-dev",
		"-n", "dev-connect",
		"port-forward",
		"service/dev-connect-gateway-dev01",
		"55221:22",
	}

	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}

func TestBuildAuthCanICommand(t *testing.T) {
	cmd := AuthCanICommand(AuthCanIOptions{
		Namespace:   "dev-connect",
		Verb:        "create",
		Resource:    "pods/portforward",
		ContextName: "platform-dev",
	})

	want := []string{
		"--context", "platform-dev",
		"-n", "dev-connect",
		"auth", "can-i",
		"create",
		"pods/portforward",
	}

	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}

func TestBuildVersionCommand(t *testing.T) {
	cmd := VersionCommand(VersionOptions{
		ContextName: "platform-dev",
		Kubeconfig:  "/tmp/kubeconfig",
	})

	want := []string{
		"--kubeconfig", "/tmp/kubeconfig",
		"--context", "platform-dev",
		"version",
	}

	if !equalStrings(cmd.Args, want) {
		t.Fatalf("args = %#v, want %#v", cmd.Args, want)
	}
}

func TestFakeRunnerRecordsCommands(t *testing.T) {
	runner := NewFakeRunner(FakeResult{Stdout: "yes\n"})

	result, err := runner.Run(context.Background(), Command{Args: []string{"version", "--client"}})
	if err != nil {
		t.Fatalf("fake runner: %v", err)
	}

	if result.Stdout != "yes\n" {
		t.Fatalf("stdout = %q, want yes", result.Stdout)
	}
	if len(runner.Commands()) != 1 {
		t.Fatalf("recorded commands = %d, want 1", len(runner.Commands()))
	}
}

func TestFakeRunnerReturnsExitError(t *testing.T) {
	runner := NewFakeRunner(FakeResult{Stderr: "denied", ExitCode: 1})

	_, err := runner.Run(context.Background(), Command{Args: []string{"auth", "can-i"}})
	if err == nil {
		t.Fatal("expected exit error")
	}

	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want ExitError", err)
	}
	if exitErr.Code != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.Code)
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
