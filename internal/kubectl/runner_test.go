package kubectl

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecutableRunnerRunsKubectlFromPath(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "stdout:$1"
echo "stderr:$2" >&2
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir)},
	}

	result, err := runner.Run(context.Background(), Command{
		Args: []string{"version", "--client"},
	})
	if err != nil {
		t.Fatalf("run kubectl: %v", err)
	}

	if strings.TrimSpace(result.Stdout) != "stdout:version" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "stderr:--client" {
		t.Fatalf("stderr = %q", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, want 0", result.ExitCode)
	}
}

func TestExecutableRunnerUsesConfiguredEnvironment(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "$HTTPS_PROXY"
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir), "HTTPS_PROXY=http://enterprise-proxy.example"},
	}

	result, err := runner.Run(context.Background(), Command{})
	if err != nil {
		t.Fatalf("run kubectl: %v", err)
	}

	if strings.TrimSpace(result.Stdout) != "http://enterprise-proxy.example" {
		t.Fatalf("stdout = %q, want proxy value", result.Stdout)
	}
}

func TestExecutableRunnerCommandEnvOverridesBaseEnv(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "$HTTPS_PROXY"
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir), "HTTPS_PROXY=http://enterprise-proxy.example"},
	}

	result, err := runner.Run(context.Background(), Command{
		Env: []string{"HTTPS_PROXY=http://override.example"},
	})
	if err != nil {
		t.Fatalf("run kubectl: %v", err)
	}

	if strings.TrimSpace(result.Stdout) != "http://override.example" {
		t.Fatalf("stdout = %q, want command override", result.Stdout)
	}
}

func TestExecutableRunnerReturnsExitError(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "denied" >&2
exit 7
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir)},
	}

	result, err := runner.Run(context.Background(), Command{})
	if err == nil {
		t.Fatal("failing kubectl succeeded")
	}

	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want ExitError", err)
	}
	if exitErr.Code != 7 {
		t.Fatalf("exit code = %d, want 7", exitErr.Code)
	}
	if !strings.Contains(exitErr.Stderr, "denied") {
		t.Fatalf("stderr = %q, want denied", exitErr.Stderr)
	}
	if result.ExitCode != 7 {
		t.Fatalf("result exit code = %d, want 7", result.ExitCode)
	}
}

func TestExecutableRunnerRunsUntilReadyAndStopsLongRunningProcess(t *testing.T) {
	binDir := t.TempDir()
	markerPath := filepath.Join(t.TempDir(), "completed")
	readyMarkerPath := filepath.Join(t.TempDir(), "ready")
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
trap 'exit 0' TERM
echo "starting"
echo "ready" > "$READY_MARKER_PATH"
echo "Forwarding from 127.0.0.1:55221 -> 22"
sleep 30
echo "completed" > "$MARKER_PATH"
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir), "MARKER_PATH=" + markerPath, "READY_MARKER_PATH=" + readyMarkerPath},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runner.RunUntilReady(ctx, Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err != nil {
		t.Fatalf("run until ready: %v", err)
	}
	if _, err := os.Stat(readyMarkerPath); err != nil {
		t.Fatalf("readiness was not reached before return; stdout=%q stderr=%q err=%v", result.Stdout, result.Stderr, err)
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("long-running process was not stopped after readiness: %v", err)
	}
}

func TestExecutableRunnerRunUntilReadyReturnsExitErrorBeforeReadiness(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "failed before ready" >&2
exit 3
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir)},
	}

	_, err := runner.RunUntilReady(context.Background(), Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err == nil {
		t.Fatal("failing kubectl succeeded")
	}

	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want ExitError", err)
	}
	if exitErr.Code != 3 {
		t.Fatalf("exit code = %d, want 3", exitErr.Code)
	}
}

func TestExecutableRunnerRunUntilReadyRequiresReadyFunc(t *testing.T) {
	runner := ExecutableRunner{Path: filepath.Join(t.TempDir(), "kubectl")}

	_, err := runner.RunUntilReady(context.Background(), Command{}, nil)
	if err == nil {
		t.Fatal("nil ready func accepted")
	}
}

func TestExecutableRunnerStartsBackgroundProcessUntilReady(t *testing.T) {
	binDir := t.TempDir()
	markerPath := filepath.Join(t.TempDir(), "completed")
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
trap 'exit 0' TERM
echo "Forwarding from 127.0.0.1:55221 -> 22"
sleep 30
echo "completed" > "$MARKER_PATH"
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir), "MARKER_PATH=" + markerPath},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	started, err := runner.StartUntilReady(ctx, Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err != nil {
		t.Fatalf("start until ready: %v", err)
	}
	if started.PID <= 0 {
		t.Fatalf("pid = %d, want positive PID", started.PID)
	}
	if !strings.Contains(started.Stdout, "Forwarding from ") {
		t.Fatalf("stdout = %q, want readiness output", started.Stdout)
	}
	process, err := os.FindProcess(started.PID)
	if err == nil {
		_ = process.Kill()
	}
	if _, err := os.Stat(markerPath); !os.IsNotExist(err) {
		t.Fatalf("background process completed unexpectedly: %v", err)
	}
}

func TestExecutableRunnerBackgroundProcessSurvivesContextCancelAfterReadiness(t *testing.T) {
	binDir := t.TempDir()
	markerPath := filepath.Join(t.TempDir(), "survived")
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
trap 'exit 0' TERM
echo "Forwarding from 127.0.0.1:55221 -> 22"
sleep 1
echo "survived" > "$MARKER_PATH"
sleep 30
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir), "MARKER_PATH=" + markerPath},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	started, err := runner.StartUntilReady(ctx, Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err != nil {
		cancel()
		t.Fatalf("start until ready: %v", err)
	}
	defer func() {
		process, findErr := os.FindProcess(started.PID)
		if findErr == nil {
			_ = process.Kill()
		}
	}()

	cancel()
	time.Sleep(1500 * time.Millisecond)
	if _, err := os.Stat(markerPath); err != nil {
		t.Fatalf("background process did not survive context cancellation after readiness: %v", err)
	}
}

func TestExecutableRunnerStartUntilReadyReturnsExitErrorBeforeReadiness(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
echo "failed before ready" >&2
exit 5
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{testPath(binDir)},
	}

	_, err := runner.StartUntilReady(context.Background(), Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err == nil {
		t.Fatal("failing kubectl succeeded")
	}
	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want ExitError", err)
	}
	if exitErr.Code != 5 {
		t.Fatalf("exit code = %d, want 5", exitErr.Code)
	}
}

func TestExecutableRunnerRequiresKubectlPath(t *testing.T) {
	runner := ExecutableRunner{Path: filepath.Join(t.TempDir(), "missing-kubectl")}

	_, err := runner.Run(context.Background(), Command{})
	if err == nil {
		t.Fatal("missing kubectl path accepted")
	}
}

func TestResolveExecutableFindsKubectlOnPath(t *testing.T) {
	binDir := t.TempDir()
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`exit 0`))

	got, err := ResolveExecutable("kubectl", []string{binDir})
	if err != nil {
		t.Fatalf("resolve executable: %v", err)
	}
	if got != kubectlPath {
		t.Fatalf("path = %q, want %q", got, kubectlPath)
	}
}

func TestResolveExecutableRejectsMissingKubectl(t *testing.T) {
	_, err := ResolveExecutable("kubectl", []string{t.TempDir()})
	if err == nil {
		t.Fatal("missing kubectl resolved")
	}
}

func TestResolveLocalExecutableUsesExplicitPathFirst(t *testing.T) {
	explicitPath := writePlainExecutable(t, "kubectl-explicit.exe")
	envPath := writePlainExecutable(t, "kubectl-env.exe")

	got, err := ResolveLocalExecutable(ResolveOptions{
		ExplicitPath: explicitPath,
		EnvPath:      envPath,
		GOOS:         "windows",
	})
	if err != nil {
		t.Fatalf("resolve kubectl: %v", err)
	}
	if got != explicitPath {
		t.Fatalf("path = %q, want explicit %q", got, explicitPath)
	}
}

func TestResolveLocalExecutableUsesConfiguredPathAfterEnv(t *testing.T) {
	envPath := writePlainExecutable(t, "kubectl-env.exe")
	configPath := writePlainExecutable(t, "kubectl-config.exe")

	got, err := ResolveLocalExecutable(ResolveOptions{
		EnvPath:        envPath,
		ConfiguredPath: configPath,
		GOOS:           "windows",
	})
	if err != nil {
		t.Fatalf("resolve kubectl: %v", err)
	}
	if got != envPath {
		t.Fatalf("path = %q, want env %q", got, envPath)
	}
}

func TestResolveLocalExecutableFindsKubectlNextToDevConnect(t *testing.T) {
	dir := t.TempDir()
	devConnectPath := filepath.Join(dir, "dev-connect.exe")
	kubectlPath := filepath.Join(dir, "kubectl.exe")
	for _, path := range []string{devConnectPath, kubectlPath} {
		if err := os.WriteFile(path, []byte("binary"), 0o600); err != nil {
			t.Fatalf("write executable %s: %v", path, err)
		}
	}

	got, err := ResolveLocalExecutable(ResolveOptions{
		DevConnectPath: devConnectPath,
		GOOS:           "windows",
	})
	if err != nil {
		t.Fatalf("resolve kubectl: %v", err)
	}
	if got != kubectlPath {
		t.Fatalf("path = %q, want adjacent kubectl %q", got, kubectlPath)
	}
}

func TestResolveLocalExecutableUsesAdditionalDefaultPath(t *testing.T) {
	defaultPath := writePlainExecutable(t, "kubectl-default.exe")

	got, err := ResolveLocalExecutable(ResolveOptions{
		GOOS:              "windows",
		AdditionalDefault: []string{defaultPath},
	})
	if err != nil {
		t.Fatalf("resolve kubectl: %v", err)
	}
	if got != defaultPath {
		t.Fatalf("path = %q, want default %q", got, defaultPath)
	}
}

func writeFakeExecutable(t *testing.T, dir, name, content string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("shell script fake executable is not used on Windows")
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write fake executable: %v", err)
	}
	return path
}

func writePlainExecutable(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte("binary"), 0o700); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	return path
}

func fakeScript(body string) string {
	return "#!/bin/sh\n" + body
}

func testPath(binDir string) string {
	path := binDir
	if existingPath := os.Getenv("PATH"); existingPath != "" {
		path += string(os.PathListSeparator) + existingPath
	}
	return "PATH=" + path
}
