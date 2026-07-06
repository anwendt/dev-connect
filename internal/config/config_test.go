package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidConfig(t *testing.T) {
	cfg, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
targets:
  dev01:
    gateway: dev01
    hostKeyRef: dev01
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if cfg.Targets["dev01"].Gateway != "dev01" {
		t.Fatalf("target gateway = %q, want dev01", cfg.Targets["dev01"].Gateway)
	}
}

func TestParseHostKeyInventory(t *testing.T) {
	cfg, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
targets:
  dev01:
    gateway: dev01
    hostKeyRef: dev01
hostKeys:
  dev01: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakePinnedHostKey dev01
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if cfg.HostKeys["dev01"] == "" {
		t.Fatal("host key inventory entry was not parsed")
	}
}

func TestParseRejectsUnknownHostKeyReferenceWhenInventoryIsConfigured(t *testing.T) {
	_, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
targets:
  dev01:
    gateway: dev01
    hostKeyRef: missing
hostKeys:
  dev02: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakePinnedHostKey dev02
`))
	if err == nil {
		t.Fatal("expected unknown host key reference error")
	}
}

func TestParseRejectsUnsupportedVersion(t *testing.T) {
	_, err := Parse([]byte(`
apiVersion: dev-connect/v0
kind: DevConnectConfig
`))
	if err == nil {
		t.Fatal("expected unsupported apiVersion error")
	}
}

func TestDefaultDirForOS(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		home    string
		appData string
		want    string
	}{
		{name: "macOS", goos: "darwin", home: "/Users/alice", want: "/Users/alice/Library/Application Support/dev-connect"},
		{name: "Linux", goos: "linux", home: "/home/alice", want: "/home/alice/.config/dev-connect"},
		{name: "Windows", goos: "windows", appData: `C:\Users\Alice\AppData\Roaming`, want: `C:\Users\Alice\AppData\Roaming/dev-connect`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DefaultDirFor(tt.goos, tt.home, tt.appData)
			if err != nil {
				t.Fatalf("DefaultDirFor: %v", err)
			}
			if got != tt.want {
				t.Fatalf("DefaultDirFor = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadUsesExplicitPathFirst(t *testing.T) {
	tempDir := t.TempDir()
	explicitPath := writeConfigFile(t, tempDir, "explicit.yaml", "explicit")
	envPath := writeConfigFile(t, tempDir, "env.yaml", "env")

	loader := Loader{
		ExplicitPath: explicitPath,
		EnvConfig:    envPath,
	}

	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Path != explicitPath {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, explicitPath)
	}
	if loaded.Config.Targets["dev01"].Gateway != "explicit" {
		t.Fatalf("gateway = %q, want explicit", loaded.Config.Targets["dev01"].Gateway)
	}
}

func TestLoadUsesEnvPathWhenExplicitPathEmpty(t *testing.T) {
	tempDir := t.TempDir()
	envPath := writeConfigFile(t, tempDir, "env.yaml", "env")

	loader := Loader{EnvConfig: envPath}

	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Path != envPath {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, envPath)
	}
}

func TestLoadUsesOSDefaultPathBeforeDevelopmentPath(t *testing.T) {
	tempDir := t.TempDir()
	defaultDir := filepath.Join(tempDir, "default")
	devDir := filepath.Join(tempDir, "dev")
	defaultPath := writeConfigFile(t, defaultDir, DefaultFileName, "default")
	devPath := writeConfigFile(t, devDir, DefaultFileName, "dev")

	loader := Loader{
		DefaultDir:     defaultDir,
		DevelopmentDir: devDir,
	}

	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Path != defaultPath {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, defaultPath)
	}
	if loaded.Config.Targets["dev01"].Gateway != "default" {
		t.Fatalf("gateway = %q, want default", loaded.Config.Targets["dev01"].Gateway)
	}
	_ = devPath
}

func TestLoadUsesDevelopmentPathWhenDefaultMissing(t *testing.T) {
	tempDir := t.TempDir()
	defaultDir := filepath.Join(tempDir, "default")
	devDir := filepath.Join(tempDir, "dev")
	devPath := writeConfigFile(t, devDir, DefaultFileName, "dev")

	loader := Loader{
		DefaultDir:     defaultDir,
		DevelopmentDir: devDir,
	}

	loaded, err := loader.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.Path != devPath {
		t.Fatalf("loaded path = %q, want %q", loaded.Path, devPath)
	}
}

func TestLoadFailsWhenNoConfigExists(t *testing.T) {
	tempDir := t.TempDir()
	loader := Loader{
		DefaultDir:     filepath.Join(tempDir, "default"),
		DevelopmentDir: filepath.Join(tempDir, "dev"),
	}

	_, err := loader.Load()
	if err == nil {
		t.Fatal("load without config succeeded, want error")
	}
}

func TestValidateRejectsTargetWithUnknownGateway(t *testing.T) {
	_, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
targets:
  dev01:
    gateway: missing
    hostKeyRef: dev01
gateways:
  dev02:
    namespace: dev-connect
    service: dev-connect-gateway-dev02
    port: 22
`))
	if err == nil {
		t.Fatal("expected unknown gateway error")
	}
}

func TestValidateRejectsContextWithUnknownCluster(t *testing.T) {
	_, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
contexts:
  platform-dev:
    cluster: missing
    gateway: dev01
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
clusters:
  dev:
    kubernetesContext: rancher-dev
`))
	if err == nil {
		t.Fatal("expected unknown cluster error")
	}
}

func TestValidateRejectsContextWithUnknownGateway(t *testing.T) {
	_, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
contexts:
  platform-dev:
    cluster: dev
    gateway: missing
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
clusters:
  dev:
    kubernetesContext: rancher-dev
`))
	if err == nil {
		t.Fatal("expected unknown gateway error")
	}
}

func TestValidatePreservesProxyOverrideWithoutEnvironmentMutation(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://enterprise-proxy.example")

	cfg, err := Parse([]byte(`
apiVersion: dev-connect/v1
kind: DevConnectConfig
clusters:
  dev:
    proxy:
      enabled: true
      httpsProxy: http://override.example
targets:
  dev01:
    gateway: dev01
    hostKeyRef: dev01
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if cfg.Clusters["dev"].Proxy.HTTPSProxy != "http://override.example" {
		t.Fatalf("proxy override not parsed: %#v", cfg.Clusters["dev"].Proxy)
	}
	if os.Getenv("HTTPS_PROXY") != "http://enterprise-proxy.example" {
		t.Fatalf("environment mutated, HTTPS_PROXY=%q", os.Getenv("HTTPS_PROXY"))
	}
}

func writeConfigFile(t *testing.T, dir, name, gateway string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}

	path := filepath.Join(dir, name)
	data := []byte(`apiVersion: dev-connect/v1
kind: DevConnectConfig
targets:
  dev01:
    gateway: ` + gateway + `
    hostKeyRef: dev01
gateways:
  ` + gateway + `:
    namespace: dev-connect
    service: dev-connect-gateway-` + gateway + `
    port: 22
`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config %s: %v", path, err)
	}
	return path
}
