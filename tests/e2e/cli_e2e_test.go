package e2e_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIConnectSkeletonEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	configPath := writeConfig(t)
	sessionDir := t.TempDir()
	sshDir := t.TempDir()
	kubectlPath := writeFakeKubectl(t, "yes")

	output := runCLI(t, binary,
		"--config", configPath,
		"--output", "json",
		"--context", "platform-dev",
		"connect", "dev01",
		"--no-code",
		"--no-reconnect",
		withEnv("DEV_CONNECT_SESSION_DIR="+sessionDir),
		withEnv("DEV_CONNECT_SSH_DIR="+sshDir),
		withEnv("DEV_CONNECT_TEST_LOCAL_PORT=55221"),
		withEnv("DEV_CONNECT_KUBECTL_PATH="+kubectlPath),
	)

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, output)
	}

	if response["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", response["apiVersion"])
	}
	if response["status"] != "Prepared" {
		t.Fatalf("status = %v, want Prepared", response["status"])
	}
	if response["server"] != "dev01" {
		t.Fatalf("server = %v, want dev01", response["server"])
	}
	if response["sessionId"] == "" {
		t.Fatal("sessionId is empty")
	}
	if response["localPort"] != float64(55221) {
		t.Fatalf("localPort = %v, want 55221", response["localPort"])
	}
	if _, err := os.Stat(filepath.Join(sessionDir, "session.json")); err != nil {
		t.Fatalf("session state was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sshDir, "ssh_config")); err != nil {
		t.Fatalf("ssh config was not written: %v", err)
	}
}

func TestCLIRejectsInvalidConfigEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dev-connect.yaml")
	if err := os.WriteFile(configPath, []byte("apiVersion: dev-connect/v0\nkind: DevConnectConfig\n"), 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	cmd := exec.Command(binary, "--config", configPath, "connect", "dev01")
	cmd.Env = append(os.Environ(), "DEV_CONNECT_CONFIG=")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("invalid config succeeded:\n%s", string(output))
	}
}

func TestCLIListTargetsEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	configPath := writeConfig(t)

	output := runCLI(t, binary, "--config", configPath, "--output", "json", "list")

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, output)
	}
	targets, ok := response["targets"].([]any)
	if !ok || len(targets) != 1 {
		t.Fatalf("targets = %#v, want one target", response["targets"])
	}
	target := targets[0].(map[string]any)
	if target["name"] != "dev01" || target["gateway"] != "dev01" {
		t.Fatalf("target = %#v, want dev01/dev01", target)
	}
}

func TestCLIStatusEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	sessionDir := t.TempDir()
	writeSessionState(t, sessionDir, "/tmp/ssh_config", "/tmp/known_hosts")

	output := runCLI(t, binary, "--output", "json", "status", withEnv("DEV_CONNECT_SESSION_DIR="+sessionDir))

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, output)
	}
	if response["status"] != "Connected" || response["server"] != "dev01" || response["sessionId"] != "session-1" {
		t.Fatalf("unexpected status response: %#v", response)
	}
}

func TestCLIDisconnectEndToEnd(t *testing.T) {
	binary := buildCLI(t)
	sessionDir := t.TempDir()
	sshDir := t.TempDir()
	sshConfigPath := filepath.Join(sshDir, "ssh_config")
	knownHostsPath := filepath.Join(sshDir, "known_hosts")
	if err := os.WriteFile(sshConfigPath, []byte("Host dev01\n"), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}
	if err := os.WriteFile(knownHostsPath, []byte("host key\n"), 0o600); err != nil {
		t.Fatalf("write known hosts: %v", err)
	}
	writeSessionState(t, sessionDir, sshConfigPath, knownHostsPath)

	output := runCLI(t, binary, "--output", "json", "disconnect", withEnv("DEV_CONNECT_SESSION_DIR="+sessionDir))

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, output)
	}
	if response["status"] != "Disconnected" {
		t.Fatalf("status = %v, want Disconnected", response["status"])
	}
	for _, path := range []string{filepath.Join(sessionDir, "session.json"), sshConfigPath, knownHostsPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or unexpected stat error: %v", path, err)
		}
	}
}

func TestCLIConfigLocationEndToEnd(t *testing.T) {
	binary := buildCLI(t)

	output := runCLI(t, binary, "config", "location")
	if output == "" {
		t.Fatal("config location output is empty")
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	binary := filepath.Join(tempDir, "dev-connect")
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, "./cmd/dev-connect")
	cmd.Dir = projectRoot(t)
	cmd.Env = append(os.Environ(), "GOCACHE="+filepath.Join(projectRoot(t), ".cache", "go-build"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build CLI: %v\n%s", err, string(output))
	}
	return binary
}

type envArg string

func withEnv(value string) envArg {
	return envArg(value)
}

func runCLI(t *testing.T, binary string, args ...any) string {
	t.Helper()

	var cliArgs []string
	env := append(os.Environ(), "DEV_CONNECT_CONFIG=")
	for _, arg := range args {
		switch value := arg.(type) {
		case string:
			cliArgs = append(cliArgs, value)
		case envArg:
			env = append(env, string(value))
		default:
			t.Fatalf("unsupported runCLI argument type %T", arg)
		}
	}

	cmd := exec.Command(binary, cliArgs...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("run dev-connect %v: %v\nstdout:\n%s\nstderr:\n%s", cliArgs, err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

func writeSessionState(t *testing.T, sessionDir, sshConfigPath, knownHostsPath string) {
	t.Helper()

	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("create session dir: %v", err)
	}
	data := []byte(`{
  "sessionId": "session-1",
  "target": "dev01",
  "gateway": "dev01",
  "namespace": "dev-connect",
  "localPort": 55221,
  "sshConfigPath": "` + sshConfigPath + `",
  "knownHostsPath": "` + knownHostsPath + `",
  "reconnect": false
}`)
	if err := os.WriteFile(filepath.Join(sessionDir, "session.json"), data, 0o600); err != nil {
		t.Fatalf("write session state: %v", err)
	}
}

func writeConfig(t *testing.T) string {
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
`)
	if err := os.WriteFile(configPath, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

func writeFakeKubectl(t *testing.T, canI string) string {
	t.Helper()

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

func projectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	return root
}
