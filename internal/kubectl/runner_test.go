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
		BaseEnv: []string{"PATH=" + binDir},
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
		BaseEnv: []string{"PATH=" + binDir, "HTTPS_PROXY=http://enterprise-proxy.example"},
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
		BaseEnv: []string{"PATH=" + binDir, "HTTPS_PROXY=http://enterprise-proxy.example"},
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
		BaseEnv: []string{"PATH=" + binDir},
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
	kubectlPath := writeFakeExecutable(t, binDir, "kubectl", fakeScript(`
trap 'exit 0' TERM
echo "starting"
echo "Forwarding from 127.0.0.1:55221 -> 22"
sleep 30
echo "completed" > "$MARKER_PATH"
`))

	runner := ExecutableRunner{
		Path:    kubectlPath,
		BaseEnv: []string{"PATH=" + binDir, "MARKER_PATH=" + markerPath},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := runner.RunUntilReady(ctx, Command{}, func(output string) bool {
		return strings.Contains(output, "Forwarding from ")
	})
	if err != nil {
		t.Fatalf("run until ready: %v", err)
	}
	if !strings.Contains(result.Stdout, "Forwarding from 127.0.0.1:55221 -> 22") {
		t.Fatalf("stdout missing readiness output: %q", result.Stdout)
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
		BaseEnv: []string{"PATH=" + binDir},
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

func fakeScript(body string) string {
	return "#!/bin/sh\n" + body
}
