package e2e_test

import (
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

	output := runCLI(t, binary,
		"--config", configPath,
		"--output", "json",
		"connect", "dev01",
		"--no-code",
		"--no-reconnect",
		withEnv("DEV_CONNECT_SESSION_DIR="+sessionDir),
		withEnv("DEV_CONNECT_SSH_DIR="+sshDir),
		withEnv("DEV_CONNECT_TEST_LOCAL_PORT=55221"),
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
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run dev-connect %v: %v\n%s", cliArgs, err, string(output))
	}
	return string(output)
}

func writeConfig(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "dev-connect.yaml")
	data := []byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
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

func projectRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve project root: %v", err)
	}
	return root
}
