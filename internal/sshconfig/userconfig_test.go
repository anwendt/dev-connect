package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultUserConfigPathForOS(t *testing.T) {
	tests := []struct {
		name        string
		goos        string
		home        string
		userProfile string
		want        string
	}{
		{name: "macOS", goos: "darwin", home: "/Users/alice", want: "/Users/alice/.ssh/config"},
		{name: "Linux", goos: "linux", home: "/home/alice", want: "/home/alice/.ssh/config"},
		{name: "Windows", goos: "windows", userProfile: `C:\Users\Alice`, want: filepath.Join(`C:\Users\Alice`, ".ssh", "config")},
		{name: "WindowsHomeFallback", goos: "windows", home: `C:\Users\Alice`, want: filepath.Join(`C:\Users\Alice`, ".ssh", "config")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DefaultUserConfigPathFor(tt.goos, tt.home, tt.userProfile)
			if err != nil {
				t.Fatalf("DefaultUserConfigPathFor: %v", err)
			}
			if got != tt.want {
				t.Fatalf("path = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUpsertManagedBlockCreatesUserSSHConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".ssh", "config")

	if err := UpsertManagedBlock(UserConfigOptions{
		ConfigPath:     configPath,
		Alias:          "dev01",
		User:           "developer",
		IdentityFile:   filepath.Join(t.TempDir(), "id dev01"),
		LocalHost:      "127.0.0.1",
		LocalPort:      55221,
		KnownHostsPath: filepath.Join(t.TempDir(), "known_hosts"),
	}); err != nil {
		t.Fatalf("upsert managed block: %v", err)
	}

	data := readFile(t, configPath)
	assertContains(t, data, "# BEGIN dev-connect dev01")
	assertContains(t, data, "Host dev01")
	assertContains(t, data, "HostName 127.0.0.1")
	assertContains(t, data, "Port 55221")
	assertContains(t, data, "User developer")
	assertContains(t, data, "IdentityFile ")
	assertContains(t, data, "UserKnownHostsFile ")
	assertContains(t, data, "StrictHostKeyChecking yes")
	assertContains(t, data, "# END dev-connect dev01")
}

func TestUpsertManagedBlockReplacesExistingManagedBlockAndPreservesUserContent(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".ssh", "config")
	initial := strings.Join([]string{
		"Host github.com",
		"  User git",
		"",
		"# BEGIN dev-connect dev01",
		"Host dev01",
		"  HostName 127.0.0.1",
		"  Port 10000",
		"# END dev-connect dev01",
		"",
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(initial), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}

	if err := UpsertManagedBlock(UserConfigOptions{
		ConfigPath:     configPath,
		Alias:          "dev01",
		LocalHost:      "127.0.0.1",
		LocalPort:      55221,
		KnownHostsPath: filepath.Join(t.TempDir(), "known_hosts"),
	}); err != nil {
		t.Fatalf("upsert managed block: %v", err)
	}

	data := readFile(t, configPath)
	if strings.Count(data, "# BEGIN dev-connect dev01") != 1 {
		t.Fatalf("managed block count changed:\n%s", data)
	}
	assertContains(t, data, "Port 55221")
	assertNotContains(t, data, "Port 10000")
	assertContains(t, data, "Host github.com")
	assertContains(t, data, "  User git")
}

func TestRemoveManagedBlockRemovesOnlyMatchingManagedBlock(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".ssh", "config")
	content := strings.Join([]string{
		"# BEGIN dev-connect dev01",
		"Host dev01",
		"  HostName 127.0.0.1",
		"# END dev-connect dev01",
		"",
		"Host github.com",
		"  User git",
		"",
		"# BEGIN dev-connect dev02",
		"Host dev02",
		"  HostName 127.0.0.1",
		"# END dev-connect dev02",
		"",
	}, "\n")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}

	if err := RemoveManagedBlock(configPath, "dev01"); err != nil {
		t.Fatalf("remove managed block: %v", err)
	}

	data := readFile(t, configPath)
	assertNotContains(t, data, "# BEGIN dev-connect dev01")
	assertContains(t, data, "Host github.com")
	assertContains(t, data, "# BEGIN dev-connect dev02")
}

func TestRemoveManagedBlockIgnoresMissingConfig(t *testing.T) {
	if err := RemoveManagedBlock(filepath.Join(t.TempDir(), ".ssh", "config"), "dev01"); err != nil {
		t.Fatalf("remove missing managed block: %v", err)
	}
}
