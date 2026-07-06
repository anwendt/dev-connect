package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestListOutputsVersionedJSON(t *testing.T) {
	stdout := executeCommand(t, "--output", "json", "list")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v, want ok", got["status"])
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
	stdout := executeCommand(t, "--config", configPath, "connect", "dev01", "--no-code", "--no-reconnect", "--output", "json")

	got := decodeJSON(t, stdout)
	if got["apiVersion"] != "v1" {
		t.Fatalf("apiVersion = %v, want v1", got["apiVersion"])
	}
	if got["status"] != "Pending" {
		t.Fatalf("status = %v, want Pending", got["status"])
	}
	if got["server"] != "dev01" {
		t.Fatalf("server = %v, want dev01", got["server"])
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

func executeCommand(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newRootCommand()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stdout)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}

	return stdout.String()
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
