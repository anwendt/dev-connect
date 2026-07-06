package vscode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// LaunchOptions describes a VS Code Remote SSH launch request.
type LaunchOptions struct {
	TargetAlias string
	UserDataDir string
	Env         []string
}

// ExitError describes a VS Code launcher process failure.
type ExitError struct {
	Code   int
	Stderr string
}

func (err ExitError) Error() string {
	return fmt.Sprintf("VS Code launcher exited with code %d: %s", err.Code, err.Stderr)
}

// Launcher starts VS Code Desktop.
type Launcher interface {
	Launch(ctx context.Context, options LaunchOptions) error
}

// ExecutableLauncher starts VS Code through its command line launcher.
type ExecutableLauncher struct {
	Path    string
	BaseEnv []string
}

// RemoteSSHArgs returns VS Code CLI arguments for a Remote SSH target.
func RemoteSSHArgs(options LaunchOptions) ([]string, error) {
	if options.TargetAlias == "" {
		return nil, errors.New("target alias is required")
	}
	args := make([]string, 0, 4)
	if options.UserDataDir != "" {
		args = append(args, "--user-data-dir", options.UserDataDir)
	}
	args = append(args, "--remote", "ssh-remote+"+options.TargetAlias)
	return args, nil
}

// Launch starts VS Code Desktop for a Remote SSH target.
func (launcher ExecutableLauncher) Launch(ctx context.Context, options LaunchOptions) error {
	if launcher.Path == "" {
		return errors.New("VS Code launcher path is required")
	}

	args, err := RemoteSSHArgs(options)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, launcher.Path, args...)
	cmd.Env = append(launcher.baseEnv(), options.Env...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return ExitError{Code: exitErr.ExitCode(), Stderr: stderr.String()}
		}
		return fmt.Errorf("launch VS Code: %w", err)
	}
	return nil
}

func (launcher ExecutableLauncher) baseEnv() []string {
	if launcher.BaseEnv != nil {
		return launcher.BaseEnv
	}
	return os.Environ()
}
