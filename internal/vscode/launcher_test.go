package vscode

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRemoteSSHArgs(t *testing.T) {
	args, err := RemoteSSHArgs(LaunchOptions{TargetAlias: "dev01"})
	if err != nil {
		t.Fatalf("remote SSH args: %v", err)
	}

	want := []string{"--remote", "ssh-remote+dev01"}
	if !equalStrings(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestRemoteSSHArgsIncludesUserDataDir(t *testing.T) {
	args, err := RemoteSSHArgs(LaunchOptions{TargetAlias: "dev01", UserDataDir: "/tmp/dev-connect/vscode"})
	if err != nil {
		t.Fatalf("remote SSH args: %v", err)
	}

	want := []string{"--user-data-dir", "/tmp/dev-connect/vscode", "--remote", "ssh-remote+dev01"}
	if !equalStrings(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestRemoteSSHArgsRejectsMissingTarget(t *testing.T) {
	_, err := RemoteSSHArgs(LaunchOptions{})
	if err == nil {
		t.Fatal("missing target alias accepted")
	}
}

func TestExitErrorFormatsMessage(t *testing.T) {
	err := ExitError{Code: 4, Stderr: "launch failed"}
	if !strings.Contains(err.Error(), "code 4") || !strings.Contains(err.Error(), "launch failed") {
		t.Fatalf("Error() = %q", err.Error())
	}
}

func TestExecutableLauncherRejectsMissingPath(t *testing.T) {
	err := ExecutableLauncher{}.Launch(context.Background(), LaunchOptions{TargetAlias: "dev01"})
	if err == nil {
		t.Fatal("missing launcher path accepted")
	}
}

func TestExecutableLauncherPropagatesMissingExecutable(t *testing.T) {
	err := ExecutableLauncher{Path: filepath.Join(t.TempDir(), "missing-code")}.Launch(context.Background(), LaunchOptions{TargetAlias: "dev01"})
	if err == nil {
		t.Fatal("missing executable launched successfully")
	}
	if !strings.Contains(err.Error(), "launch VS Code") {
		t.Fatalf("error = %q, want launch VS Code", err)
	}
}

func TestExecutableLauncherRunsVSCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake launcher is not used on Windows")
	}

	outputPath := filepath.Join(t.TempDir(), "args.txt")
	launcherPath := writeFakeLauncher(t, `
printf '%s\n' "$@" > "$OUTPUT_PATH"
`)

	launcher := ExecutableLauncher{
		Path:    launcherPath,
		BaseEnv: []string{"OUTPUT_PATH=" + outputPath},
	}

	if err := launcher.Launch(context.Background(), LaunchOptions{TargetAlias: "dev01"}); err != nil {
		t.Fatalf("launch VS Code: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read launcher output: %v", err)
	}
	if strings.TrimSpace(string(data)) != "--remote\nssh-remote+dev01" {
		t.Fatalf("launcher args = %q", string(data))
	}
}

func TestExecutableLauncherReturnsExitError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake launcher is not used on Windows")
	}

	launcher := ExecutableLauncher{
		Path: writeFakeLauncher(t, `
echo "launch failed" >&2
exit 4
`),
	}

	err := launcher.Launch(context.Background(), LaunchOptions{TargetAlias: "dev01"})
	if err == nil {
		t.Fatal("failing launcher succeeded")
	}

	var exitErr ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %T, want ExitError", err)
	}
	if exitErr.Code != 4 {
		t.Fatalf("exit code = %d, want 4", exitErr.Code)
	}
	if !strings.Contains(exitErr.Stderr, "launch failed") {
		t.Fatalf("stderr = %q, want launch failed", exitErr.Stderr)
	}
}

func writeFakeLauncher(t *testing.T, body string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "code")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o700); err != nil {
		t.Fatalf("write fake launcher: %v", err)
	}
	return path
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
