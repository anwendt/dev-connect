package tunnel

import (
	"context"
	"errors"
	"testing"

	"github.com/anwendt/dev-connect/internal/kubectl"
)

func TestOutputIndicatesReadiness(t *testing.T) {
	output := "Forwarding from 127.0.0.1:55221 -> 22\n"
	if !IsReadyOutput(output) {
		t.Fatalf("output not ready: %q", output)
	}
}

func TestSupervisorStartsPortForwardWithFakeKubectl(t *testing.T) {
	runner := kubectl.NewFakeRunner(kubectl.FakeResult{
		Stdout: "Forwarding from 127.0.0.1:55221 -> 22\n",
	})
	supervisor := Supervisor{Runner: runner}

	result, err := supervisor.Start(context.Background(), StartOptions{
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
		Reconnect:  true,
	})
	if err != nil {
		t.Fatalf("start tunnel: %v", err)
	}

	if !result.Ready {
		t.Fatal("tunnel result not ready")
	}
	if len(runner.Commands()) != 1 {
		t.Fatalf("commands = %d, want 1", len(runner.Commands()))
	}
}

func TestSupervisorStartsBackgroundPortForwardWhenSupported(t *testing.T) {
	runner := &fakeBackgroundRunner{
		started: kubectl.StartedProcess{
			PID:    4242,
			Stdout: "Forwarding from 127.0.0.1:55221 -> 22\n",
		},
	}
	supervisor := Supervisor{Runner: runner}

	result, err := supervisor.Start(context.Background(), StartOptions{
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
		Reconnect:  true,
	})
	if err != nil {
		t.Fatalf("start tunnel: %v", err)
	}

	if !result.Ready || result.PID != 4242 {
		t.Fatalf("result = %#v, want ready PID 4242", result)
	}
	if !runner.startCalled {
		t.Fatal("StartUntilReady was not called")
	}
}

func TestSupervisorReconnectsOnceAfterTransientFailure(t *testing.T) {
	runner := kubectl.NewFakeRunner(
		kubectl.FakeResult{Stderr: "temporary failure", ExitCode: 1},
		kubectl.FakeResult{Stdout: "Forwarding from 127.0.0.1:55221 -> 22\n"},
	)
	supervisor := Supervisor{Runner: runner, MaxReconnects: 1}

	result, err := supervisor.Start(context.Background(), StartOptions{
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
		Reconnect:  true,
	})
	if err != nil {
		t.Fatalf("start tunnel with reconnect: %v", err)
	}

	if !result.Ready || result.Attempts != 2 {
		t.Fatalf("result = %#v, want ready after 2 attempts", result)
	}
}

type fakeBackgroundRunner struct {
	started     kubectl.StartedProcess
	startCalled bool
}

func (runner *fakeBackgroundRunner) Run(context.Context, kubectl.Command) (kubectl.Result, error) {
	return kubectl.Result{}, errors.New("Run should not be called")
}

func (runner *fakeBackgroundRunner) StartUntilReady(_ context.Context, _ kubectl.Command, ready kubectl.ReadyFunc) (kubectl.StartedProcess, error) {
	runner.startCalled = true
	if !ready(runner.started.Stdout) {
		return kubectl.StartedProcess{}, errors.New("not ready")
	}
	return runner.started, nil
}

func TestSupervisorDoesNotReconnectWhenDisabled(t *testing.T) {
	runner := kubectl.NewFakeRunner(kubectl.FakeResult{Stderr: "failure", ExitCode: 1})
	supervisor := Supervisor{Runner: runner, MaxReconnects: 1}

	_, err := supervisor.Start(context.Background(), StartOptions{
		Namespace:  "dev-connect",
		Service:    "dev-connect-gateway-dev01",
		LocalPort:  55221,
		RemotePort: 22,
		Reconnect:  false,
	})
	if err == nil {
		t.Fatal("start without reconnect succeeded, want error")
	}

	var exitErr kubectl.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want kubectl.ExitError", err)
	}
	if len(runner.Commands()) != 1 {
		t.Fatalf("commands = %d, want 1", len(runner.Commands()))
	}
}
