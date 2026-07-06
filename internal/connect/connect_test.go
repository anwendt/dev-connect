package connect

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/anwendt/dev-connect/internal/config"
	"github.com/anwendt/dev-connect/internal/port"
	"github.com/anwendt/dev-connect/internal/session"
)

const pinnedHostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakePinnedHostKey dev01"

func TestPrepareCreatesSSHFilesAndSessionState(t *testing.T) {
	startedAt := time.Unix(1700000000, 0).UTC()
	result, err := Prepare(Options{
		Config:     testConfig(),
		TargetName: "dev01",
		Context:    "platform-dev",
		SessionDir: t.TempDir(),
		SSHDir:     t.TempDir(),
		AllocatePort: func() (port.Allocation, error) {
			return port.Allocation{Host: "127.0.0.1", Port: 55221}, nil
		},
		Now:       func() time.Time { return startedAt },
		SessionID: func() string { return "session-1" },
		Reconnect: true,
	})
	if err != nil {
		t.Fatalf("prepare connect session: %v", err)
	}

	if result.TargetName != "dev01" {
		t.Fatalf("target = %q, want dev01", result.TargetName)
	}
	if result.GatewayName != "gw-dev01" {
		t.Fatalf("gateway = %q, want gw-dev01", result.GatewayName)
	}
	if result.Gateway.Service != "dev-connect-gateway-dev01" {
		t.Fatalf("gateway service = %q", result.Gateway.Service)
	}
	if result.LocalPort != 55221 {
		t.Fatalf("local port = %d, want 55221", result.LocalPort)
	}

	configData := readText(t, result.SSHFiles.ConfigPath)
	knownHostsData := readText(t, result.SSHFiles.KnownHostsPath)
	assertContains(t, configData, "Host dev01")
	assertContains(t, configData, "HostName 127.0.0.1")
	assertContains(t, configData, "Port 55221")
	assertContains(t, configData, "User developer")
	assertContains(t, configData, "StrictHostKeyChecking yes")
	assertContains(t, knownHostsData, "[127.0.0.1]:55221 "+pinnedHostKey)

	state, err := session.Store{Dir: result.SessionDir}.Load()
	if err != nil {
		t.Fatalf("load session state: %v", err)
	}
	if state.SessionID != "session-1" || state.Target != "dev01" || state.LocalPort != 55221 {
		t.Fatalf("unexpected session state: %#v", state)
	}
	if state.Gateway != "gw-dev01" || state.Namespace != "dev-connect" {
		t.Fatalf("unexpected gateway state: %#v", state)
	}
	if state.KubernetesContext != "rancher-dev" {
		t.Fatalf("kubernetes context = %q, want rancher-dev", state.KubernetesContext)
	}
	if state.SSHConfigPath != result.SSHFiles.ConfigPath || state.KnownHostsPath != result.SSHFiles.KnownHostsPath {
		t.Fatalf("session SSH paths = %q/%q, want %q/%q", state.SSHConfigPath, state.KnownHostsPath, result.SSHFiles.ConfigPath, result.SSHFiles.KnownHostsPath)
	}
	if !state.Reconnect || !state.StartedAt.Equal(startedAt) {
		t.Fatalf("unexpected session timing/reconnect state: %#v", state)
	}
}

func TestPrepareUsesExplicitGatewayOverride(t *testing.T) {
	cfg := testConfig()
	cfg.Gateways["override"] = config.Gateway{Namespace: "dev-connect", Service: "dev-connect-gateway-override", Port: 22}

	result, err := Prepare(Options{
		Config:      cfg,
		TargetName:  "dev01",
		GatewayName: "override",
		SessionDir:  t.TempDir(),
		SSHDir:      t.TempDir(),
		AllocatePort: func() (port.Allocation, error) {
			return port.Allocation{Host: "127.0.0.1", Port: 55222}, nil
		},
		SessionID: func() string { return "session-override" },
	})
	if err != nil {
		t.Fatalf("prepare with gateway override: %v", err)
	}
	if result.GatewayName != "override" || result.Gateway.Service != "dev-connect-gateway-override" {
		t.Fatalf("gateway override not used: %#v", result)
	}
}

func TestPrepareRejectsUnknownTarget(t *testing.T) {
	_, err := Prepare(Options{
		Config:     testConfig(),
		TargetName: "missing",
		SessionDir: t.TempDir(),
		SSHDir:     t.TempDir(),
	})
	if err == nil {
		t.Fatal("unknown target accepted")
	}
}

func TestPrepareRejectsMissingPinnedHostKey(t *testing.T) {
	cfg := testConfig()
	delete(cfg.HostKeys, "dev01")

	_, err := Prepare(Options{
		Config:     cfg,
		TargetName: "dev01",
		SessionDir: t.TempDir(),
		SSHDir:     t.TempDir(),
	})
	if err == nil {
		t.Fatal("missing host key accepted")
	}
}

func TestPrepareReturnsPortAllocationErrorBeforeWritingState(t *testing.T) {
	sessionDir := t.TempDir()
	_, err := Prepare(Options{
		Config:     testConfig(),
		TargetName: "dev01",
		SessionDir: sessionDir,
		SSHDir:     t.TempDir(),
		AllocatePort: func() (port.Allocation, error) {
			return port.Allocation{}, errors.New("no free port")
		},
	})
	if err == nil {
		t.Fatal("port allocation error ignored")
	}

	if _, statErr := os.Stat(filepath.Join(sessionDir, "session.json")); !os.IsNotExist(statErr) {
		t.Fatalf("session state was written after allocation failure: %v", statErr)
	}
}

func testConfig() config.Config {
	return config.Config{
		APIVersion: config.APIVersion,
		Kind:       config.Kind,
		Contexts: map[string]config.Context{
			"platform-dev": {
				Cluster: "dev",
				Gateway: "gw-dev01",
			},
		},
		Clusters: map[string]config.Cluster{
			"dev": {
				KubernetesContext: "rancher-dev",
			},
		},
		Gateways: map[string]config.Gateway{
			"gw-dev01": {
				Namespace: "dev-connect",
				Service:   "dev-connect-gateway-dev01",
				Port:      22,
			},
		},
		Targets: map[string]config.Target{
			"dev01": {
				Gateway:    "gw-dev01",
				User:       "developer",
				HostKeyRef: "dev01",
			},
		},
		HostKeys: map[string]string{
			"dev01": pinnedHostKey,
		},
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, data, want string) {
	t.Helper()

	if !strings.Contains(data, want) {
		t.Fatalf("data missing %q:\n%s", want, data)
	}
}
