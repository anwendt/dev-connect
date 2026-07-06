// Package main provides the dev-connect CLI entry point.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anwendt/dev-connect/internal/config"
	"github.com/anwendt/dev-connect/internal/connect"
	"github.com/anwendt/dev-connect/internal/kubectl"
	"github.com/anwendt/dev-connect/internal/output"
	"github.com/anwendt/dev-connect/internal/port"
	"github.com/anwendt/dev-connect/internal/preflight"
	"github.com/anwendt/dev-connect/internal/proxy"
	"github.com/anwendt/dev-connect/internal/session"
	"github.com/anwendt/dev-connect/internal/sshconfig"
	"github.com/anwendt/dev-connect/internal/vscode"
)

type cliOptions struct {
	configPath  string
	contextName string
	clusterName string
	gatewayName string
	logLevel    string
	logFormat   string
	output      string
	noCode      bool
	noReconnect bool
	timeout     time.Duration
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	opts := cliOptions{}
	cmd := &cobra.Command{
		Use:   "dev-connect",
		Short: "Connect VS Code Desktop to private-cloud development servers through Kubernetes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Path to dev-connect YAML configuration")
	cmd.PersistentFlags().StringVar(&opts.contextName, "context", "", "dev-connect context name")
	cmd.PersistentFlags().StringVar(&opts.clusterName, "cluster", "", "dev-connect cluster name")
	cmd.PersistentFlags().StringVar(&opts.gatewayName, "gateway", "", "dev-connect gateway name")
	cmd.PersistentFlags().StringVar(&opts.logLevel, "log-level", "info", "Log level: error, warn, info, debug")
	cmd.PersistentFlags().StringVar(&opts.logFormat, "log-format", "text", "Log format: text, json")
	cmd.PersistentFlags().StringVar(&opts.output, "output", "text", "Command output format: text, json")
	cmd.PersistentFlags().BoolVar(&opts.noCode, "no-code", false, "Do not launch VS Code Desktop")
	cmd.PersistentFlags().BoolVar(&opts.noReconnect, "no-reconnect", false, "Disable automatic tunnel reconnect")
	cmd.PersistentFlags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "Command timeout")

	cmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		return validateOptions(opts)
	}

	cmd.AddCommand(newConnectCommand(&opts))
	cmd.AddCommand(newDisconnectCommand(&opts))
	cmd.AddCommand(newStatusCommand(&opts))
	cmd.AddCommand(newListCommand(&opts))
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newConfigCommand())
	helpCommand := newHelpCommand(cmd)
	cmd.AddCommand(helpCommand)
	cmd.SetHelpCommand(helpCommand)

	return cmd
}

func validateOptions(opts cliOptions) error {
	if !oneOf(opts.output, "text", "json") {
		return fmt.Errorf("invalid argument %q for --output: expected text or json", opts.output)
	}
	if !oneOf(opts.logFormat, "text", "json") {
		return fmt.Errorf("invalid argument %q for --log-format: expected text or json", opts.logFormat)
	}
	if !oneOf(opts.logLevel, "error", "warn", "info", "debug") {
		return fmt.Errorf("invalid argument %q for --log-level: expected error, warn, info, or debug", opts.logLevel)
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func newConnectCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "connect <target>",
		Short: "Prepare a VS Code Remote SSH connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			loaded, err := loadConfig(*opts)
			if err != nil {
				return err
			}

			sessionDir, err := sessionDir()
			if err != nil {
				return err
			}
			sshDir, err := sshDir(sessionDir)
			if err != nil {
				return err
			}

			cluster, err := selectedCluster(loaded.Config, *opts)
			if err != nil {
				return err
			}
			result, err := connect.Prepare(connect.Options{
				Config:       loaded.Config,
				TargetName:   args[0],
				Context:      opts.contextName,
				GatewayName:  opts.gatewayName,
				SessionDir:   sessionDir,
				SSHDir:       sshDir,
				AllocatePort: allocatePort,
				Reconnect:    !opts.noReconnect,
			})
			if err != nil {
				return err
			}

			if err := runPreflight(cmd.Context(), *opts, cluster, result); err != nil {
				_ = sshconfig.Cleanup(result.SSHFiles)
				_ = (session.Store{Dir: result.SessionDir}).Clear()
				return err
			}

			if !opts.noCode {
				if err := launchVSCode(cmd.Context(), loaded.Config, args[0]); err != nil {
					_ = sshconfig.Cleanup(result.SSHFiles)
					_ = (session.Store{Dir: result.SessionDir}).Clear()
					return err
				}
			}

			response := output.Response{
				Status:    "Prepared",
				Server:    args[0],
				SessionID: result.State.SessionID,
				LocalPort: result.LocalPort,
			}
			return writeResponse(cmd, opts, response)
		},
	}
}

func launchVSCode(ctx context.Context, cfg config.Config, targetAlias string) error {
	if path := os.Getenv("DEV_CONNECT_CODE_PATH"); path != "" {
		return vscode.ExecutableLauncher{Path: path}.Launch(ctx, vscode.LaunchOptions{TargetAlias: targetAlias})
	}
	path, err := vscode.ResolveLauncher(vscode.Options{
		LocalAppData:   os.Getenv("LOCALAPPDATA"),
		ConfiguredPath: cfg.VSCode.LauncherPath,
	})
	if err != nil {
		return err
	}
	return vscode.ExecutableLauncher{Path: path}.Launch(ctx, vscode.LaunchOptions{TargetAlias: targetAlias})
}

func runPreflight(ctx context.Context, opts cliOptions, cluster config.Cluster, result connect.Result) error {
	if opts.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
	}

	kubectlPath, err := kubectlPath()
	if err != nil {
		return err
	}

	runner := kubectl.ExecutableRunner{
		Path: kubectlPath,
		BaseEnv: proxy.BuildEnv(os.Environ(), proxy.Config{
			Enabled:    cluster.Proxy.Enabled,
			HTTPProxy:  cluster.Proxy.HTTPProxy,
			HTTPSProxy: cluster.Proxy.HTTPSProxy,
			NoProxy:    cluster.Proxy.NoProxy,
		}),
	}

	_, err = preflight.Validate(ctx, preflight.Options{
		Runner:      runner,
		Namespace:   result.Gateway.Namespace,
		Service:     result.Gateway.Service,
		LocalPort:   result.LocalPort,
		RemotePort:  result.Gateway.Port,
		ContextName: cluster.KubernetesContext,
		Kubeconfig:  cluster.Kubeconfig,
	})
	return err
}

func selectedCluster(cfg config.Config, opts cliOptions) (config.Cluster, error) {
	if opts.clusterName != "" {
		cluster, ok := cfg.Clusters[opts.clusterName]
		if !ok {
			return config.Cluster{}, fmt.Errorf("unknown cluster %q", opts.clusterName)
		}
		return cluster, nil
	}
	if opts.contextName == "" {
		return config.Cluster{}, nil
	}
	contextConfig, ok := cfg.Contexts[opts.contextName]
	if !ok {
		return config.Cluster{}, fmt.Errorf("unknown context %q", opts.contextName)
	}
	cluster, ok := cfg.Clusters[contextConfig.Cluster]
	if !ok {
		return config.Cluster{}, fmt.Errorf("context %q references unknown cluster %q", opts.contextName, contextConfig.Cluster)
	}
	return cluster, nil
}

func kubectlPath() (string, error) {
	if path := os.Getenv("DEV_CONNECT_KUBECTL_PATH"); path != "" {
		return kubectl.ResolveExecutable(path, nil)
	}
	return kubectl.ResolveExecutable("kubectl", strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)))
}

func sessionDir() (string, error) {
	if dir := os.Getenv("DEV_CONNECT_SESSION_DIR"); dir != "" {
		return dir, nil
	}
	dir, err := config.DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session"), nil
}

func sshDir(sessionDir string) (string, error) {
	if dir := os.Getenv("DEV_CONNECT_SSH_DIR"); dir != "" {
		return dir, nil
	}
	if sessionDir == "" {
		return "", fmt.Errorf("session directory is required")
	}
	return filepath.Join(sessionDir, "ssh"), nil
}

func allocatePort() (port.Allocation, error) {
	if value := os.Getenv("DEV_CONNECT_TEST_LOCAL_PORT"); value != "" {
		localPort, err := strconv.Atoi(value)
		if err != nil {
			return port.Allocation{}, fmt.Errorf("parse DEV_CONNECT_TEST_LOCAL_PORT: %w", err)
		}
		return port.Allocation{Host: "127.0.0.1", Port: localPort}, nil
	}
	return port.AllocateLoopback()
}

func loadConfig(opts cliOptions) (config.Loaded, error) {
	loader := config.Loader{
		ExplicitPath: opts.configPath,
		EnvConfig:    os.Getenv("DEV_CONNECT_CONFIG"),
	}
	return loader.Load()
}

func newDisconnectCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "disconnect",
		Short: "Disconnect the managed dev-connect session",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionDir, err := sessionDir()
			if err != nil {
				return err
			}
			store := session.Store{Dir: sessionDir}
			state, err := store.Load()
			if err != nil && !errors.Is(err, session.ErrNotFound) {
				return err
			}
			if err == nil {
				if cleanupErr := sshconfig.Cleanup(sshconfig.SessionFiles{
					ConfigPath:     state.SSHConfigPath,
					KnownHostsPath: state.KnownHostsPath,
				}); cleanupErr != nil {
					return cleanupErr
				}
			}
			if err := store.Clear(); err != nil {
				return err
			}
			return writeResponse(cmd, opts, output.Response{Status: "Disconnected"})
		},
	}
}

func newStatusCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Print managed session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			sessionDir, err := sessionDir()
			if err != nil {
				return err
			}
			state, err := (session.Store{Dir: sessionDir}).Load()
			if errors.Is(err, session.ErrNotFound) {
				return writeResponse(cmd, opts, output.Response{Status: "Disconnected"})
			}
			if err != nil {
				return err
			}
			return writeResponse(cmd, opts, output.Response{
				Status:    "Connected",
				Server:    state.Target,
				SessionID: state.SessionID,
				LocalPort: state.LocalPort,
			})
		},
	}
}

func newListCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured dev-connect targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			loaded, err := loadConfig(*opts)
			if err != nil {
				return err
			}
			return writeResponse(cmd, opts, output.Response{
				Status:  "ok",
				Targets: targetSummaries(loaded.Config),
			})
		},
	}
}

func targetSummaries(cfg config.Config) []output.Target {
	names := make([]string, 0, len(cfg.Targets))
	for name := range cfg.Targets {
		names = append(names, name)
	}
	sort.Strings(names)

	targets := make([]output.Target, 0, len(names))
	for _, name := range names {
		target := cfg.Targets[name]
		targets = append(targets, output.Target{
			Name:    name,
			Gateway: target.Gateway,
			User:    target.User,
		})
	}
	return targets
}

func newHelpCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "help",
		Short: "Print help information",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return root.Help()
		},
	}
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return output.WriteJSON(cmd.OutOrStdout(), output.Response{
				APIVersion: output.APIVersion,
				Status:     "ok",
			})
		},
	}
}

func writeResponse(cmd *cobra.Command, opts *cliOptions, response output.Response) error {
	if opts.output == "json" {
		return output.WriteJSON(cmd.OutOrStdout(), response)
	}

	if response.Server != "" {
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", response.Status, response.Server)
		return err
	}
	if len(response.Targets) > 0 {
		for _, target := range response.Targets {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), target.Name); err != nil {
				return err
			}
		}
		return nil
	}
	_, err := fmt.Fprintln(cmd.OutOrStdout(), response.Status)
	return err
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage dev-connect configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "location",
		Short: "Print the default configuration directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := config.DefaultDir()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), dir)
			return err
		},
	})

	return cmd
}
