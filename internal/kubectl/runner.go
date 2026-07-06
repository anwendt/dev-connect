package kubectl

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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

// RunUntilReady executes kubectl until output indicates readiness, then stops the process.
func (runner ExecutableRunner) RunUntilReady(ctx context.Context, command Command, ready ReadyFunc) (Result, error) {
	if ready == nil {
		return Result{}, errors.New("ready function is required")
	}
	kubectlPath := runner.Path
	if kubectlPath == "" {
		return Result{}, errors.New("kubectl path is required")
	}

	cmd := exec.CommandContext(ctx, kubectlPath, command.Args...)
	cmd.Env = mergeEnv(runner.baseEnv(), command.Env)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, fmt.Errorf("open kubectl stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return Result{}, fmt.Errorf("open kubectl stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("start kubectl: %w", err)
	}

	events := make(chan outputEvent, 2)
	go scanOutput(stdoutPipe, streamStdout, events)
	go scanOutput(stderrPipe, streamStderr, events)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	for {
		select {
		case event := <-events:
			if event.err != nil {
				continue
			}
			switch event.stream {
			case streamStdout:
				stdout.WriteString(event.line)
				stdout.WriteByte('\n')
			case streamStderr:
				stderr.WriteString(event.line)
				stderr.WriteByte('\n')
			}
			if ready(event.line) {
				stopProcess(cmd)
				<-waitCh
				return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: 0}, nil
			}
		case err := <-waitCh:
			result := Result{
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				ExitCode: exitCode(err),
			}
			if err != nil {
				if result.ExitCode >= 0 {
					return result, ExitError{Code: result.ExitCode, Stderr: result.Stderr}
				}
				return result, fmt.Errorf("run kubectl until ready: %w", err)
			}
			return result, nil
		case <-ctx.Done():
			stopProcess(cmd)
			<-waitCh
			return Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1}, ctx.Err()
		}
	}
}

// StartUntilReady starts kubectl and leaves it running after readiness is detected.
func (runner ExecutableRunner) StartUntilReady(ctx context.Context, command Command, ready ReadyFunc) (StartedProcess, error) {
	if ready == nil {
		return StartedProcess{}, errors.New("ready function is required")
	}
	kubectlPath := runner.Path
	if kubectlPath == "" {
		return StartedProcess{}, errors.New("kubectl path is required")
	}

	cmd := exec.Command(kubectlPath, command.Args...)
	cmd.Env = mergeEnv(runner.baseEnv(), command.Env)

	stdoutFile, err := os.CreateTemp("", "dev-connect-kubectl-stdout-*.log")
	if err != nil {
		return StartedProcess{}, fmt.Errorf("open kubectl stdout log: %w", err)
	}
	defer func() {
		_ = stdoutFile.Close()
	}()
	stderrFile, err := os.CreateTemp("", "dev-connect-kubectl-stderr-*.log")
	if err != nil {
		return StartedProcess{}, fmt.Errorf("open kubectl stderr log: %w", err)
	}
	defer func() {
		_ = stderrFile.Close()
	}()
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	if err := cmd.Start(); err != nil {
		return StartedProcess{}, fmt.Errorf("start kubectl: %w", err)
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stdout := readFileBestEffort(stdoutFile.Name())
			stderr := readFileBestEffort(stderrFile.Name())
			if readyOutput(stdout, ready) || readyOutput(stderr, ready) {
				return StartedProcess{
					PID:    cmd.Process.Pid,
					Stdout: stdout,
					Stderr: stderr,
				}, nil
			}
		case err := <-waitCh:
			stdout := readFileBestEffort(stdoutFile.Name())
			stderr := readFileBestEffort(stderrFile.Name())
			result := Result{
				Stdout:   stdout,
				Stderr:   stderr,
				ExitCode: exitCode(err),
			}
			if readyOutput(stdout, ready) || readyOutput(stderr, ready) {
				return StartedProcess{
					PID:    cmd.Process.Pid,
					Stdout: stdout,
					Stderr: stderr,
				}, nil
			}
			if err != nil {
				if result.ExitCode >= 0 {
					return StartedProcess{}, ExitError{Code: result.ExitCode, Stderr: result.Stderr}
				}
				return StartedProcess{}, fmt.Errorf("start kubectl until ready: %w", err)
			}
			return StartedProcess{}, errors.New("kubectl exited before readiness")
		case <-ctx.Done():
			stopProcess(cmd)
			<-waitCh
			return StartedProcess{}, ctx.Err()
		}
	}
}

func readFileBestEffort(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

func readyOutput(output string, ready ReadyFunc) bool {
	for _, line := range strings.Split(output, "\n") {
		if ready(line) {
			return true
		}
	}
	return false
}

func executablePath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if runtime.GOOS == "windows" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".exe", ".cmd", ".bat", ".com":
			return path, nil
		default:
			return "", fmt.Errorf("%s is not a Windows executable", path)
		}
	}
	if info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("%s is not executable", path)
	}
	return path, nil
}

type streamName string

const (
	streamStdout streamName = "stdout"
	streamStderr streamName = "stderr"
)

type outputEvent struct {
	stream streamName
	line   string
	err    error
}

func scanOutput(reader io.Reader, stream streamName, events chan<- outputEvent) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		events <- outputEvent{stream: stream, line: scanner.Text()}
	}
	if err := scanner.Err(); err != nil {
		events <- outputEvent{stream: stream, err: err}
	}
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
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
