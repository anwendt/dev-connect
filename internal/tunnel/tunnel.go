package tunnel

import (
	"context"
	"errors"
	"strings"

	"github.com/anwendt/dev-connect/internal/kubectl"
)

// StartOptions describes a managed port-forward request.
type StartOptions struct {
	Namespace   string
	Service     string
	LocalPort   int
	RemotePort  int
	ContextName string
	Kubeconfig  string
	Reconnect   bool
}

// Result describes the supervised tunnel startup outcome.
type Result struct {
	Ready    bool
	Attempts int
	PID      int
}

// Supervisor starts kubectl port-forward through an injected runner.
type Supervisor struct {
	Runner        kubectl.Runner
	MaxReconnects int
}

// Start starts a managed port-forward command.
func (supervisor Supervisor) Start(ctx context.Context, options StartOptions) (Result, error) {
	if supervisor.Runner == nil {
		return Result{}, errors.New("kubectl runner is required")
	}

	attemptsAllowed := 1
	if options.Reconnect {
		attemptsAllowed += supervisor.MaxReconnects
	}
	if attemptsAllowed < 1 {
		attemptsAllowed = 1
	}

	command := kubectl.PortForwardServiceCommand(kubectl.PortForwardOptions{
		Namespace:   options.Namespace,
		Service:     options.Service,
		LocalPort:   options.LocalPort,
		RemotePort:  options.RemotePort,
		ContextName: options.ContextName,
		Kubeconfig:  options.Kubeconfig,
	})

	var lastErr error
	for attempt := 1; attempt <= attemptsAllowed; attempt++ {
		if backgroundRunner, ok := supervisor.Runner.(kubectl.BackgroundRunner); ok {
			started, err := backgroundRunner.StartUntilReady(ctx, command, IsReadyOutput)
			if err == nil {
				return Result{Ready: true, Attempts: attempt, PID: started.PID}, nil
			}
			lastErr = err
			continue
		}
		result, err := supervisor.Runner.Run(ctx, command)
		if err == nil {
			return Result{Ready: IsReadyOutput(result.Stdout), Attempts: attempt}, nil
		}
		lastErr = err
	}

	return Result{Attempts: attemptsAllowed}, lastErr
}

// IsReadyOutput reports whether kubectl output indicates port-forward readiness.
func IsReadyOutput(output string) bool {
	return strings.Contains(output, "Forwarding from ")
}
