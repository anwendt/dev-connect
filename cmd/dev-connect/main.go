// Package main provides the dev-connect CLI entry point.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/anwendt/dev-connect/internal/config"
	"github.com/anwendt/dev-connect/internal/output"
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
			if _, err := loadConfig(*opts); err != nil {
				return err
			}
			response := output.Response{
				Status: "Pending",
				Server: args[0],
			}
			return writeResponse(cmd, opts, response)
		},
	}
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
			return writeResponse(cmd, opts, output.Response{Status: "Disconnected"})
		},
	}
}

func newListCommand(opts *cliOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured dev-connect targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeResponse(cmd, opts, output.Response{Status: "ok"})
		},
	}
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
