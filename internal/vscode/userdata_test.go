package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareUserDataDirWritesRemoteSSHConfigSetting(t *testing.T) {
	dir := t.TempDir()
	sshConfigPath := filepath.Join(t.TempDir(), "ssh_config")

	if err := PrepareUserDataDir(dir, sshConfigPath); err != nil {
		t.Fatalf("prepare user data dir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "User", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var settings map[string]string
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("decode settings: %v\n%s", err, string(data))
	}
	if settings["remote.SSH.configFile"] != sshConfigPath {
		t.Fatalf("remote.SSH.configFile = %q, want %q", settings["remote.SSH.configFile"], sshConfigPath)
	}
}

func TestPrepareUserDataDirRejectsMissingSSHConfigPath(t *testing.T) {
	if err := PrepareUserDataDir(t.TempDir(), ""); err == nil {
		t.Fatal("missing SSH config path accepted")
	}
}

func TestCleanupUserDataDirRemovesDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := PrepareUserDataDir(dir, filepath.Join(t.TempDir(), "ssh_config")); err != nil {
		t.Fatalf("prepare user data dir: %v", err)
	}
	if err := CleanupUserDataDir(dir); err != nil {
		t.Fatalf("cleanup user data dir: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("user data dir still exists or unexpected stat error: %v", err)
	}
}
