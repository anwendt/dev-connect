package kubectl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExecutableRunner executes the locally installed kubectl binary.
type ExecutableRunner struct {
	Path    string
	BaseEnv []string
}

// ResolveExecutable locates a kubectl executable in the supplied search paths.
func ResolveExecutable(name string, searchPaths []string) (string, error) {
	if name == "" {
		return "", errors.New("executable name is required")
	}
	if filepath.IsAbs(name) {
		return executablePath(name)
	}
	for _, dir := range searchPaths {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if runtime.GOOS == "windows" && filepath.Ext(candidate) == "" {
			candidate += ".exe"
		}
		path, err := executablePath(candidate)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s executable not found", name)
}

// Run executes kubectl with the supplied arguments and process environment.
func (runner ExecutableRunner) Run(ctx context.Context, command Command) (Result, error) {
	kubectlPath := runner.Path
	if kubectlPath == "" {
		return Result{}, errors.New("kubectl path is required")
	}

	cmd := exec.CommandContext(ctx, kubectlPath, command.Args...)
	cmd.Env = mergeEnv(runner.baseEnv(), command.Env)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode(err),
	}
	if err != nil {
		if result.ExitCode >= 0 {
			return result, ExitError{Code: result.ExitCode, Stderr: result.Stderr}
		}
		return result, fmt.Errorf("run kubectl: %w", err)
	}
	return result, nil
}

func executablePath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable", path)
	}
	return path, nil
}

func (runner ExecutableRunner) baseEnv() []string {
	if runner.BaseEnv != nil {
		return runner.BaseEnv
	}
	return os.Environ()
}

func mergeEnv(base, overrides []string) []string {
	env := append([]string(nil), base...)
	for _, override := range overrides {
		key, _, ok := strings.Cut(override, "=")
		if !ok {
			env = append(env, override)
			continue
		}
		prefix := key + "="
		replaced := false
		for i, existing := range env {
			if strings.HasPrefix(existing, prefix) {
				env[i] = override
				replaced = true
				break
			}
		}
		if !replaced {
			env = append(env, override)
		}
	}
	return env
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
