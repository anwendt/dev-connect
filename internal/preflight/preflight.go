package preflight

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anwendt/dev-connect/internal/kubectl"
	"github.com/anwendt/dev-connect/internal/tunnel"
)

var (
	// ErrPermissionDenied indicates kubectl auth can-i did not approve port-forward access.
	ErrPermissionDenied = errors.New("kubectl port-forward permission denied")
	// ErrPortForwardNotReady indicates functional port-forward validation did not reach readiness.
	ErrPortForwardNotReady = errors.New("kubectl port-forward validation did not become ready")
)

// Options describes Kubernetes preflight validation.
type Options struct {
	Runner      kubectl.Runner
	Namespace   string
	Service     string
	LocalPort   int
	RemotePort  int
	ContextName string
	Kubeconfig  string
}

// Result reports completed preflight checks.
type Result struct {
	ConnectivityOK bool
	RBACOK         bool
	FunctionalOK   bool
}

// Validate verifies kubectl API reachability, RBAC, and functional port-forward behavior.
func Validate(ctx context.Context, options Options) (Result, error) {
	if options.Runner == nil {
		return Result{}, errors.New("kubectl runner is required")
	}

	if _, err := options.Runner.Run(ctx, kubectl.VersionCommand(kubectl.VersionOptions{
		ContextName: options.ContextName,
		Kubeconfig:  options.Kubeconfig,
	})); err != nil {
		return Result{}, fmt.Errorf("verify kubectl connectivity: %w", err)
	}

	rbac, err := options.Runner.Run(ctx, kubectl.AuthCanICommand(kubectl.AuthCanIOptions{
		Namespace:   options.Namespace,
		Verb:        "create",
		Resource:    "pods/portforward",
		ContextName: options.ContextName,
		Kubeconfig:  options.Kubeconfig,
	}))
	if err != nil {
		return Result{ConnectivityOK: true}, fmt.Errorf("verify kubectl port-forward RBAC: %w", err)
	}
	if strings.TrimSpace(rbac.Stdout) != "yes" {
		return Result{ConnectivityOK: true}, ErrPermissionDenied
	}

	portForwardCommand := kubectl.PortForwardServiceCommand(kubectl.PortForwardOptions{
		Namespace:   options.Namespace,
		Service:     options.Service,
		LocalPort:   options.LocalPort,
		RemotePort:  options.RemotePort,
		ContextName: options.ContextName,
		Kubeconfig:  options.Kubeconfig,
	})
	functional, err := runPortForwardProbe(ctx, options.Runner, portForwardCommand)
	if err != nil {
		return Result{ConnectivityOK: true, RBACOK: true}, fmt.Errorf("verify kubectl port-forward functionally: %w", err)
	}
	if !tunnel.IsReadyOutput(functional.Stdout) {
		return Result{ConnectivityOK: true, RBACOK: true}, ErrPortForwardNotReady
	}

	return Result{ConnectivityOK: true, RBACOK: true, FunctionalOK: true}, nil
}

func runPortForwardProbe(ctx context.Context, runner kubectl.Runner, command kubectl.Command) (kubectl.Result, error) {
	if readyRunner, ok := runner.(kubectl.ReadyRunner); ok {
		return readyRunner.RunUntilReady(ctx, command, tunnel.IsReadyOutput)
	}
	return runner.Run(ctx, command)
}
