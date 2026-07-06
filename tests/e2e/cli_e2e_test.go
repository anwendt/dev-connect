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

	output := runCLI(t, binary,
		"--config", configPath,
		"--output", "json",
		"connect", "dev01",
		"--no-code",
		"--no-reconnect",
	)

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output: %v\n%s", err, output)
	}

	if response["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", response["apiVersion"])
	}
	if response["status"] != "Pending" {
		t.Fatalf("status = %v, want Pending", response["status"])
	}
	if response["server"] != "dev01" {
		t.Fatalf("server = %v, want dev01", response["server"])
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

func runCLI(t *testing.T, binary string, args ...string) string {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "DEV_CONNECT_CONFIG=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run dev-connect %v: %v\n%s", args, err, string(output))
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
