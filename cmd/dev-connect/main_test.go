package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/anwendt/dev-connect/internal/session"
)

func TestRootCommandIncludesApprovedCommands(t *testing.T) {
	cmd := newRootCommand()
	want := []string{
		"connect",
		"disconnect",
		"status",
		"list",
		"version",
		"help",
		"config",
	}

	for _, name := range want {
		if _, _, err := cmd.Find([]string{name}); err != nil {
			t.Fatalf("command %q missing: %v", name, err)
		}
	}
}

func TestRootCommandIncludesApprovedGlobalFlags(t *testing.T) {
	cmd := newRootCommand()
	want := []string{
		"config",
		"context",
		"cluster",
		"gateway",
		"log-level",
		"log-format",
		"output",
		"no-code",
		"no-reconnect",
		"timeout",
	}

	for _, name := range want {
		if cmd.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("persistent flag %q missing", name)
		}
	}
}

func TestVersionOutputsVersionedJSON(t *testing.T) {
	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute version: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("version output is not JSON: %v\n%s", err, stdout.String())
	}

	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
}

func TestStatusOutputsVersionedJSON(t *testing.T) {
	stdout := executeCommand(t, "status", "--output", "json")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "Disconnected" {
		t.Fatalf("status = %v, want Disconnected", got["status"])
	}
}

func TestStatusReportsActiveSession(t *testing.T) {
	sessionDir := t.TempDir()
	writeSessionState(t, sessionDir, "/tmp/ssh_config", "/tmp/known_hosts")
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)

	stdout := executeCommand(t, "status", "--output", "json")
	got := decodeJSON(t, stdout)
	if got["status"] != "Connected" {
		t.Fatalf("status = %v, want Connected", got["status"])
	}
	if got["server"] != "dev01" {
		t.Fatalf("server = %v, want dev01", got["server"])
	}
	if got["sessionId"] != "session-1" {
		t.Fatalf("sessionId = %v, want session-1", got["sessionId"])
	}
	if got["localPort"] != float64(55221) {
		t.Fatalf("localPort = %v, want 55221", got["localPort"])
	}
}

func TestDisconnectRemovesSessionAndSSHArtifacts(t *testing.T) {
	sessionDir := t.TempDir()
	sshConfigPath := filepath.Join(t.TempDir(), "ssh_config")
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	vscodeUserDataDir := filepath.Join(t.TempDir(), "vscode-user-data")
	if err := os.WriteFile(sshConfigPath, []byte("Host dev01\n"), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}
	if err := os.WriteFile(knownHostsPath, []byte("host key\n"), 0o600); err != nil {
		t.Fatalf("write known hosts: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(vscodeUserDataDir, "User"), 0o700); err != nil {
		t.Fatalf("create VS Code user data dir: %v", err)
	}
	writeSessionStateWithVSCodeUserDataDir(t, sessionDir, sshConfigPath, knownHostsPath, vscodeUserDataDir)
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)

	stdout := executeCommand(t, "disconnect", "--output", "json")
	got := decodeJSON(t, stdout)
	if got["status"] != "Disconnected" {
		t.Fatalf("status = %v, want Disconnected", got["status"])
	}
	for _, path := range []string{filepath.Join(sessionDir, "session.json"), sshConfigPath, knownHostsPath, vscodeUserDataDir} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or unexpected stat error: %v", path, err)
		}
	}
}

func TestDisconnectWithoutSessionIsIdempotent(t *testing.T) {
	t.Setenv("DEV_CONNECT_SESSION_DIR", t.TempDir())

	stdout := executeCommand(t, "disconnect", "--output", "json")
	got := decodeJSON(t, stdout)
	if got["status"] != "Disconnected" {
		t.Fatalf("status = %v, want Disconnected", got["status"])
	}
}

func TestListOutputsVersionedJSON(t *testing.T) {
	configPath := writeCLIConfig(t)
	stdout := executeCommand(t, "--config", configPath, "--output", "json", "list")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v, want ok", got["status"])
	}
	targets, ok := got["targets"].([]any)
	if !ok || len(targets) != 1 {
		t.Fatalf("targets = %#v, want one target", got["targets"])
	}
	target, ok := targets[0].(map[string]any)
	if !ok {
		t.Fatalf("target shape = %#v", targets[0])
	}
	if target["name"] != "dev01" || target["gateway"] != "dev01" {
		t.Fatalf("target = %#v, want dev01/dev01", target)
	}
}

func writeSessionState(t *testing.T, sessionDir, sshConfigPath, knownHostsPath string) {
	t.Helper()

	writeSessionStateWithPID(t, sessionDir, sshConfigPath, knownHostsPath, 0)
}

func writeSessionStateWithPID(t *testing.T, sessionDir, sshConfigPath, knownHostsPath string, portForwardPID int) {
	t.Helper()

	writeSessionStateData(t, sessionDir, sshConfigPath, knownHostsPath, "", portForwardPID)
}

func writeSessionStateWithVSCodeUserDataDir(t *testing.T, sessionDir, sshConfigPath, knownHostsPath, vscodeUserDataDir string) {
	t.Helper()

	writeSessionStateData(t, sessionDir, sshConfigPath, knownHostsPath, vscodeUserDataDir, 0)
}

func writeSessionStateData(t *testing.T, sessionDir, sshConfigPath, knownHostsPath, vscodeUserDataDir string, portForwardPID int) {
	t.Helper()

	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("create session dir: %v", err)
	}
	state := session.State{
		SessionID:         "session-1",
		Target:            "dev01",
		Gateway:           "dev01",
		Namespace:         "dev-connect",
		LocalPort:         55221,
		PortForwardPID:    portForwardPID,
		SSHConfigPath:     sshConfigPath,
		KnownHostsPath:    knownHostsPath,
		VSCodeUserDataDir: vscodeUserDataDir,
		Reconnect:         false,
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal session state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "session.json"), data, 0o600); err != nil {
		t.Fatalf("write session state: %v", err)
	}
}

func TestConnectRequiresTarget(t *testing.T) {
	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"connect"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("connect without target succeeded, want error")
	}
}

func TestConnectSkeletonAcceptsTargetWithoutSideEffects(t *testing.T) {
	configPath := writeCLIConfig(t)
	sessionDir := t.TempDir()
	sshDir := t.TempDir()
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)
	t.Setenv("DEV_CONNECT_SSH_DIR", sshDir)
	t.Setenv("DEV_CONNECT_TEST_LOCAL_PORT", "55221")
	t.Setenv("DEV_CONNECT_KUBECTL_PATH", writeFakeKubectl(t, "yes"))

	stdout := executeCommand(t, "--config", configPath, "--context", "platform-dev", "connect", "dev01", "--no-code", "--no-reconnect", "--output", "json")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "Prepared" {
		t.Fatalf("status = %v, want Prepared", got["status"])
	}
	if got["server"] != "dev01" {
		t.Fatalf("server = %v, want dev01", got["server"])
	}
	if got["sessionId"] == "" {
		t.Fatal("sessionId is empty")
	}
	if got["localPort"] != float64(55221) {
		t.Fatalf("localPort = %v, want 55221", got["localPort"])
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "session.json")); err != nil {
		t.Fatalf("session state was not written: %v", err)
	}
	stateData, err := os.ReadFile(filepath.Join(sessionDir, "session.json"))
	if err != nil {
		t.Fatalf("read session state: %v", err)
	}
	if !strings.Contains(string(stateData), `"portForwardPid":`) {
		t.Fatalf("session state does not contain portForwardPid:\n%s", string(stateData))
	}
	if _, err := os.Stat(filepath.Join(sshDir, "ssh_config")); err != nil {
		t.Fatalf("ssh config was not written: %v", err)
	}
}

func TestConnectFailsWhenKubectlRBACDenied(t *testing.T) {
	configPath := writeCLIConfig(t)
	sessionDir := t.TempDir()
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)
	t.Setenv("DEV_CONNECT_SSH_DIR", t.TempDir())
	t.Setenv("DEV_CONNECT_TEST_LOCAL_PORT", "55221")
	t.Setenv("DEV_CONNECT_KUBECTL_PATH", writeFakeKubectl(t, "no"))

	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--config", configPath, "--context", "platform-dev", "connect", "dev01", "--no-code", "--no-reconnect"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("connect with denied kubectl RBAC succeeded")
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "session.json")); !os.IsNotExist(err) {
		t.Fatalf("session state was not cleaned after preflight failure: %v", err)
	}
}

func TestConnectLaunchesVSCodeWhenNoCodeIsFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake VS Code launcher is not used on Windows")
	}

	launcherOutput := filepath.Join(t.TempDir(), "vscode-args.txt")
	launcherPath := writeFakeCodeLauncher(t, launcherOutput)
	configPath := writeCLIConfigWithVSCodeLauncher(t, launcherPath)
	sessionDir := t.TempDir()
	t.Setenv("DEV_CONNECT_CODE_PATH", launcherPath)
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)
	t.Setenv("DEV_CONNECT_SSH_DIR", t.TempDir())
	t.Setenv("DEV_CONNECT_TEST_LOCAL_PORT", "55221")
	t.Setenv("DEV_CONNECT_KUBECTL_PATH", writeFakeKubectl(t, "yes"))

	_ = executeCommand(t, "--config", configPath, "--context", "platform-dev", "connect", "dev01", "--no-reconnect", "--output", "json")

	data, err := os.ReadFile(launcherOutput)
	if err != nil {
		t.Fatalf("read fake VS Code launcher output: %v", err)
	}
	wantArgs := "--user-data-dir\n" + filepath.Join(sessionDir, "vscode-user-data") + "\n--remote\nssh-remote+dev01"
	if strings.TrimSpace(string(data)) != wantArgs {
		t.Fatalf("VS Code launcher args = %q", string(data))
	}

	settingsPath := filepath.Join(sessionDir, "vscode-user-data", "User", "settings.json")
	settingsData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read VS Code settings: %v", err)
	}
	var settings map[string]string
	if err := json.Unmarshal(settingsData, &settings); err != nil {
		t.Fatalf("decode VS Code settings: %v\n%s", err, string(settingsData))
	}
	if settings["remote.SSH.configFile"] != filepath.Join(os.Getenv("DEV_CONNECT_SSH_DIR"), "ssh_config") {
		t.Fatalf("remote.SSH.configFile = %q", settings["remote.SSH.configFile"])
	}
}

func TestConnectRejectsExistingSessionState(t *testing.T) {
	configPath := writeCLIConfig(t)
	sessionDir := t.TempDir()
	writeSessionState(t, sessionDir, "/tmp/ssh_config", "/tmp/known_hosts")
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)
	t.Setenv("DEV_CONNECT_SSH_DIR", t.TempDir())
	t.Setenv("DEV_CONNECT_TEST_LOCAL_PORT", "55221")
	t.Setenv("DEV_CONNECT_KUBECTL_PATH", writeFakeKubectl(t, "yes"))

	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--config", configPath, "--context", "platform-dev", "connect", "dev01", "--no-code", "--no-reconnect"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("connect with existing session succeeded")
	}
}

func TestConnectClearsStaleSessionState(t *testing.T) {
	configPath := writeCLIConfig(t)
	sessionDir := t.TempDir()
	sshConfigPath := filepath.Join(t.TempDir(), "ssh_config")
	knownHostsPath := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(sshConfigPath, []byte("Host dev01\n"), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}
	if err := os.WriteFile(knownHostsPath, []byte("dev01 ssh-ed25519 key\n"), 0o600); err != nil {
		t.Fatalf("write known hosts: %v", err)
	}
	writeSessionStateWithPID(t, sessionDir, sshConfigPath, knownHostsPath, 424242)
	t.Setenv("DEV_CONNECT_SESSION_DIR", sessionDir)
	t.Setenv("DEV_CONNECT_SSH_DIR", t.TempDir())
	t.Setenv("DEV_CONNECT_TEST_LOCAL_PORT", "55221")
	t.Setenv("DEV_CONNECT_KUBECTL_PATH", writeFakeKubectl(t, "yes"))

	previousProcessExists := sessionProcessExists
	sessionProcessExists = func(pid int) bool { return pid != 424242 }
	t.Cleanup(func() {
		sessionProcessExists = previousProcessExists
	})

	stdout := executeCommand(t, "--config", configPath, "--context", "platform-dev", "connect", "dev01", "--no-code", "--no-reconnect", "--output", "json")

	got := decodeJSON(t, stdout)
	if got["status"] != "Prepared" {
		t.Fatalf("status = %v, want Prepared", got["status"])
	}
	for _, path := range []string{sshConfigPath, knownHostsPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("stale artifact %s still exists or unexpected stat error: %v", path, err)
		}
	}
}

func TestConnectFailsBeforeSideEffectsWhenConfigInvalid(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dev-connect.yaml")
	if err := os.WriteFile(configPath, []byte(`
apiVersion: dev-connect/v0
kind: DevConnectConfig
`), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--config", configPath, "connect", "dev01"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("connect with invalid config succeeded, want error")
	}
}

func TestLogFormatAndOutputFlagsAreIndependent(t *testing.T) {
	stdout := executeCommand(t, "--log-format", "text", "--output", "json", "status")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
}

func TestLifecycleMetadataLogsUseStderr(t *testing.T) {
	stdout, stderr := executeCommandStreams(t, "--log-format", "json", "--output", "json", "status")

	got := decodeJSON(t, stdout)
	if got["status"] != "Disconnected" {
		t.Fatalf("status = %v, want Disconnected", got["status"])
	}

	var logRecord map[string]any
	if err := json.Unmarshal([]byte(stderr), &logRecord); err != nil {
		t.Fatalf("stderr log is not JSON: %v\n%s", err, stderr)
	}
	if logRecord["msg"] != "dev-connect command completed" {
		t.Fatalf("log msg = %v, want command completed", logRecord["msg"])
	}
	if logRecord["command"] != "status" || logRecord["status"] != "Disconnected" {
		t.Fatalf("unexpected log record: %#v", logRecord)
	}
	if strings.Contains(stdout, "dev-connect command completed") {
		t.Fatalf("stdout contains log data:\n%s", stdout)
	}
}

func TestConfigLocationCommandPrintsDirectory(t *testing.T) {
	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"config", "location"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute config location: %v", err)
	}

	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatal("config location output is empty")
	}
}

func TestConfigValidateOutputsVersionedJSON(t *testing.T) {
	configPath := writeCLIConfig(t)
	stdout := executeCommand(t, "--config", configPath, "--output", "json", "config", "validate")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "Valid" {
		t.Fatalf("status = %v, want Valid", got["status"])
	}
}

func TestConfigValidateRejectsInvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dev-connect.yaml")
	if err := os.WriteFile(configPath, []byte("apiVersion: dev-connect/v0\nkind: DevConnectConfig\n"), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	cmd := newRootCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--config", configPath, "config", "validate"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("invalid config validation succeeded")
	}
}

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	stdout, _ := executeCommandStreams(t, args...)
	return stdout
}

func executeCommandStreams(t *testing.T, args ...string) (string, string) {
	t.Helper()

	cmd := newRootCommand()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v\nstderr:\n%s", args, err, stderr.String())
	}

	return stdout.String(), stderr.String()
}

func decodeJSON(t *testing.T, data string) map[string]any {
	t.Helper()

	var got map[string]any
	if err := json.Unmarshal([]byte(data), &got); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, data)
	}
	return got
}

func TestInvalidOutputFlagFails(t *testing.T) {
	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs([]string{"--output", "xml", "status"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("invalid output flag succeeded, want error")
	}
	if !strings.Contains(fmt.Sprint(err), "invalid argument") {
		t.Fatalf("error = %v, want invalid argument", err)
	}
}

func writeCLIConfig(t *testing.T) string {
	t.Helper()

	return writeCLIConfigData(t, "")
}

func writeCLIConfigWithVSCodeLauncher(t *testing.T, launcherPath string) string {
	t.Helper()

	return writeCLIConfigData(t, `
vscode:
  launcherPath: `+launcherPath+`
`)
}

func writeCLIConfigData(t *testing.T, suffix string) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dev-connect.yaml")
	data := []byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
contexts:
  platform-dev:
    cluster: dev
    gateway: dev01
clusters:
  dev:
    kubernetesContext: rancher-dev
targets:
  dev01:
    gateway: dev01
    hostKeyRef: dev01
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
hostKeys:
  dev01: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakePinnedHostKey dev01
` + suffix)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeFakeCodeLauncher(t *testing.T, outputPath string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "code")
	script := `#!/bin/sh
printf '%s\n' "$@" > "` + outputPath + `"
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake VS Code launcher: %v", err)
	}
	return path
}

func writeFakeKubectl(t *testing.T, canI string) string {
	t.Helper()

	if runtime.GOOS == "windows" {
		path := filepath.Join(t.TempDir(), "kubectl.cmd")
		script := `@echo off
set args=%*
echo %args% | findstr /C:"version" >nul
if %errorlevel%==0 (
  echo Client Version: v1.30.0
  echo Server Version: v1.30.0
  exit /b 0
)
echo %args% | findstr /C:"auth can-i create pods/portforward" >nul
if %errorlevel%==0 (
  echo ` + canI + `
  exit /b 0
)
echo %args% | findstr /C:"port-forward" >nul
if %errorlevel%==0 (
  echo Forwarding from 127.0.0.1:55221 -^> 22
  ping -n 6 127.0.0.1 >nul
  exit /b 0
)
echo unexpected kubectl args: %args% 1>&2
exit /b 9
`
		if err := os.WriteFile(path, []byte(script), 0o600); err != nil {
			t.Fatalf("write fake kubectl: %v", err)
		}
		return path
	}

	path := filepath.Join(t.TempDir(), "kubectl")
	script := `#!/bin/sh
case "$*" in
  *"version"*)
    echo "Client Version: v1.30.0"
    echo "Server Version: v1.30.0"
    ;;
  *"auth can-i create pods/portforward"*)
    echo "` + canI + `"
    ;;
  *"port-forward"*)
    echo "Forwarding from 127.0.0.1:55221 -> 22"
    sleep 5
    ;;
  *)
    echo "unexpected kubectl args: $*" >&2
    exit 9
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake kubectl: %v", err)
	}
	return path
}
