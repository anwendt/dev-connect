package preflight

import (
	"context"
	"errors"
	"testing"

	"github.com/anwendt/dev-connect/internal/kubectl"
)

func TestValidateRunsConnectivityRBACAndFunctionalPortForwardChecks(t *testing.T) {
	runner := kubectl.NewFakeRunner(
		kubectl.FakeResult{Stdout: "Client Version: v1.30.0\nServer Version: v1.30.0\n"},
		kubectl.FakeResult{Stdout: "yes\n"},
		kubectl.FakeResult{Stdout: "Forwarding from 127.0.0.1:55221 -> 22\n"},
	)

	result, err := Validate(context.Background(), Options{
		Runner:      runner,
		Namespace:   "dev-connect",
		Service:     "dev-connect-gateway-dev01",
		LocalPort:   55221,
		RemotePort:  22,
		ContextName: "platform-dev",
		Kubeconfig:  "/tmp/kubeconfig",
	})
	if err != nil {
		t.Fatalf("validate preflight: %v", err)
	}
	if !result.ConnectivityOK || !result.RBACOK || !result.FunctionalOK {
		t.Fatalf("unexpected result: %#v", result)
	}

	commands := runner.Commands()
	if len(commands) != 3 {
		t.Fatalf("commands = %d, want 3", len(commands))
	}
	assertArgs(t, commands[0].Args, []string{"--kubeconfig", "/tmp/kubeconfig", "--context", "platform-dev", "version"})
	assertArgs(t, commands[1].Args, []string{"--kubeconfig", "/tmp/kubeconfig", "--context", "platform-dev", "-n", "dev-connect", "auth", "can-i", "create", "pods/portforward"})
	assertArgs(t, commands[2].Args, []string{"--kubeconfig", "/tmp/kubeconfig", "--context", "platform-dev", "-n", "dev-connect", "port-forward", "service/dev-connect-gateway-dev01", "55221:22"})
}

func TestValidateUsesReadyRunnerForFunctionalPortForwardCheck(t *testing.T) {
	runner := &fakeReadyRunner{
		results: []kubectl.Result{
			{Stdout: "Client Version: v1.30.0\nServer Version: v1.30.0\n"},
			{Stdout: "yes\n"},
		},
		readyResult: kubectl.Result{Stdout: "Forwarding from 127.0.0.1:55221 -> 22\n"},
	}

	result, err := Validate(context.Background(), Options{
		Runner:     runner,
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
	})
	if err != nil {
		t.Fatalf("validate preflight: %v", err)
	}
	if !result.FunctionalOK {
		t.Fatalf("functional check not ok: %#v", result)
	}
	if !runner.readyCalled {
		t.Fatal("RunUntilReady was not called")
	}
}

func TestValidateRejectsDeniedRBACEvenWhenKubectlExitsZero(t *testing.T) {
	runner := kubectl.NewFakeRunner(
		kubectl.FakeResult{Stdout: "Client Version: v1.30.0\nServer Version: v1.30.0\n"},
		kubectl.FakeResult{Stdout: "no\n"},
	)

	_, err := Validate(context.Background(), Options{
		Runner:     runner,
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
	})
	if err == nil {
		t.Fatal("denied RBAC accepted")
	}
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("error = %v, want ErrPermissionDenied", err)
	}
	if len(runner.Commands()) != 2 {
		t.Fatalf("commands = %d, want no functional check after denied RBAC", len(runner.Commands()))
	}
}

func TestValidateRejectsPortForwardOutputWithoutReadiness(t *testing.T) {
	runner := kubectl.NewFakeRunner(
		kubectl.FakeResult{Stdout: "Client Version: v1.30.0\nServer Version: v1.30.0\n"},
		kubectl.FakeResult{Stdout: "yes\n"},
		kubectl.FakeResult{Stdout: "waiting\n"},
	)

	_, err := Validate(context.Background(), Options{
		Runner:     runner,
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
	})
	if err == nil {
		t.Fatal("non-ready port-forward output accepted")
	}
	if !errors.Is(err, ErrPortForwardNotReady) {
		t.Fatalf("error = %v, want ErrPortForwardNotReady", err)
	}
}

func TestValidateRequiresRunner(t *testing.T) {
	_, err := Validate(context.Background(), Options{})
	if err == nil {
		t.Fatal("missing runner accepted")
	}
}

type fakeReadyRunner struct {
	results     []kubectl.Result
	commands    []kubectl.Command
	readyCalled bool
	readyResult kubectl.Result
}

func (runner *fakeReadyRunner) Run(_ context.Context, command kubectl.Command) (kubectl.Result, error) {
	runner.commands = append(runner.commands, command)
	if len(runner.results) == 0 {
		return kubectl.Result{}, nil
	}
	result := runner.results[0]
	runner.results = runner.results[1:]
	return result, nil
}

func (runner *fakeReadyRunner) RunUntilReady(_ context.Context, command kubectl.Command, ready kubectl.ReadyFunc) (kubectl.Result, error) {
	runner.commands = append(runner.commands, command)
	runner.readyCalled = true
	if !ready(runner.readyResult.Stdout) {
		return runner.readyResult, nil
	}
	return runner.readyResult, nil
}

func assertArgs(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("args length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q\nall args: %#v", i, got[i], want[i], got)
		}
	}
}
