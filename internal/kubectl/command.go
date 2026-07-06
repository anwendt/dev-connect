package kubectl

import (
	"context"
	"fmt"
)

// Command describes a kubectl process invocation.
type Command struct {
	Args []string
	Env  []string
}

// Result describes completed kubectl process output.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Runner executes kubectl commands.
type Runner interface {
	Run(ctx context.Context, command Command) (Result, error)
}

// ExitError describes a kubectl command that exited unsuccessfully.
type ExitError struct {
	Code   int
	Stderr string
}

func (err ExitError) Error() string {
	return fmt.Sprintf("kubectl exited with code %d: %s", err.Code, err.Stderr)
}

// PortForwardOptions describes a kubectl service port-forward command.
type PortForwardOptions struct {
	Namespace   string
	Service     string
	LocalPort   int
	RemotePort  int
	ContextName string
	Kubeconfig  string
}

// AuthCanIOptions describes a kubectl auth can-i command.
type AuthCanIOptions struct {
	Namespace   string
	Verb        string
	Resource    string
	ContextName string
	Kubeconfig  string
}

// PortForwardServiceCommand builds a kubectl service port-forward command.
func PortForwardServiceCommand(options PortForwardOptions) Command {
	args := baseArgs(options.Kubeconfig, options.ContextName, options.Namespace)
	args = append(args,
		"port-forward",
		"service/"+options.Service,
		fmt.Sprintf("%d:%d", options.LocalPort, options.RemotePort),
	)
	return Command{Args: args}
}

// AuthCanICommand builds a kubectl auth can-i command.
func AuthCanICommand(options AuthCanIOptions) Command {
	args := baseArgs(options.Kubeconfig, options.ContextName, options.Namespace)
	args = append(args,
		"auth",
		"can-i",
		options.Verb,
		options.Resource,
	)
	return Command{Args: args}
}

func baseArgs(kubeconfig, contextName, namespace string) []string {
	args := []string{}
	if kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	if contextName != "" {
		args = append(args, "--context", contextName)
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	return args
}
