package sshconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const (
	hostKeyV1 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeHostKeyV1 dev01"
	hostKeyV2 = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeHostKeyV2 dev01"
)

func TestWriteSessionFilesPinsHostKey(t *testing.T) {
	dir := t.TempDir()

	files, err := WriteSessionFiles(SessionOptions{
		Dir:       dir,
		Alias:     "dev01",
		User:      "developer",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	configData := readFile(t, files.ConfigPath)
	knownHostsData := readFile(t, files.KnownHostsPath)

	assertContains(t, configData, "Host dev01")
	assertContains(t, configData, "HostName 127.0.0.1")
	assertContains(t, configData, "Port 55221")
	assertContains(t, configData, "User developer")
	assertContains(t, configData, "StrictHostKeyChecking yes")
	assertContains(t, configData, "UserKnownHostsFile "+quoteConfigValue(files.KnownHostsPath))
	assertNotContains(t, configData, "StrictHostKeyChecking no")
	assertNotContains(t, configData, "IdentityFile")
	assertNotContains(t, configData, "PRIVATE KEY")

	assertContains(t, knownHostsData, "[127.0.0.1]:55221 "+hostKeyV1)
}

func TestWriteSessionFilesIncludesIdentityFileWhenConfigured(t *testing.T) {
	identityFile := filepath.Join(t.TempDir(), "Application Support", "keys", "dev01")
	files, err := WriteSessionFiles(SessionOptions{
		Dir:          t.TempDir(),
		Alias:        "dev01",
		User:         "developer",
		LocalHost:    "127.0.0.1",
		LocalPort:    55221,
		HostKey:      hostKeyV1,
		IdentityFile: identityFile,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	configData := readFile(t, files.ConfigPath)
	assertContains(t, configData, "IdentityFile "+quoteConfigValue(identityFile))
	assertContains(t, configData, "IdentitiesOnly yes")
}

func TestWriteSessionFilesQuotesKnownHostsPathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Application Support", "dev-connect", "session", "ssh")

	files, err := WriteSessionFiles(SessionOptions{
		Dir:       dir,
		Alias:     "dev01",
		User:      "developer",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	configData := readFile(t, files.ConfigPath)
	assertContains(t, configData, "UserKnownHostsFile "+quoteConfigValue(files.KnownHostsPath))
	assertNotContains(t, configData, "UserKnownHostsFile "+files.KnownHostsPath+"\n")
}

func TestWriteSessionFilesRejectsMissingHostKey(t *testing.T) {
	_, err := WriteSessionFiles(SessionOptions{
		Dir:       t.TempDir(),
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
	})
	if err == nil {
		t.Fatal("missing host key accepted")
	}
}

func TestWriteSessionFilesRejectsMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		options SessionOptions
		want    string
	}{
		{
			name: "directory",
			options: SessionOptions{
				Alias:     "dev01",
				LocalHost: "127.0.0.1",
				LocalPort: 55221,
				HostKey:   hostKeyV1,
			},
			want: "directory",
		},
		{
			name: "alias",
			options: SessionOptions{
				Dir:       t.TempDir(),
				LocalHost: "127.0.0.1",
				LocalPort: 55221,
				HostKey:   hostKeyV1,
			},
			want: "alias",
		},
		{
			name: "local host",
			options: SessionOptions{
				Dir:       t.TempDir(),
				Alias:     "dev01",
				LocalPort: 55221,
				HostKey:   hostKeyV1,
			},
			want: "local host",
		},
		{
			name: "local port",
			options: SessionOptions{
				Dir:       t.TempDir(),
				Alias:     "dev01",
				LocalHost: "127.0.0.1",
				HostKey:   hostKeyV1,
			},
			want: "invalid local port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := WriteSessionFiles(tt.options)
			if err == nil {
				t.Fatal("WriteSessionFiles accepted invalid options")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestWriteSessionFilesWithoutUserOmitsUserLine(t *testing.T) {
	files, err := WriteSessionFiles(SessionOptions{
		Dir:       t.TempDir(),
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	configData := readFile(t, files.ConfigPath)
	assertNotContains(t, configData, "  User ")
}

func TestWriteSessionFilesRotatesHostKey(t *testing.T) {
	dir := t.TempDir()

	first, err := WriteSessionFiles(SessionOptions{
		Dir:       dir,
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write first session files: %v", err)
	}

	second, err := WriteSessionFiles(SessionOptions{
		Dir:       dir,
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV2,
	})
	if err != nil {
		t.Fatalf("write rotated session files: %v", err)
	}

	if first.KnownHostsPath != second.KnownHostsPath {
		t.Fatalf("known_hosts path changed: %q != %q", first.KnownHostsPath, second.KnownHostsPath)
	}

	knownHostsData := readFile(t, second.KnownHostsPath)
	assertContains(t, knownHostsData, hostKeyV2)
	assertNotContains(t, knownHostsData, hostKeyV1)
}

func TestCleanupRemovesSessionFiles(t *testing.T) {
	files, err := WriteSessionFiles(SessionOptions{
		Dir:       t.TempDir(),
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	if err := Cleanup(files); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	for _, path := range []string{files.ConfigPath, files.KnownHostsPath} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or unexpected stat error: %v", path, err)
		}
	}
}

func TestCleanupIgnoresMissingFilesAndEmptyPaths(t *testing.T) {
	if err := Cleanup(SessionFiles{
		ConfigPath:     filepath.Join(t.TempDir(), "missing_ssh_config"),
		KnownHostsPath: "",
	}); err != nil {
		t.Fatalf("cleanup missing files: %v", err)
	}
}

func TestSessionFilesUseRestrictivePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not reliable on Windows")
	}

	files, err := WriteSessionFiles(SessionOptions{
		Dir:       t.TempDir(),
		Alias:     "dev01",
		LocalHost: "127.0.0.1",
		LocalPort: 55221,
		HostKey:   hostKeyV1,
	})
	if err != nil {
		t.Fatalf("write session files: %v", err)
	}

	for _, path := range []string{files.ConfigPath, files.KnownHostsPath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if mode := info.Mode().Perm(); mode != 0o600 {
			t.Fatalf("%s mode = %v, want 0600", filepath.Base(path), mode)
		}
	}
}

func readFile(t *testing.T, path string) string {
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

func assertNotContains(t *testing.T, data, forbidden string) {
	t.Helper()

	if strings.Contains(data, forbidden) {
		t.Fatalf("data contains forbidden %q:\n%s", forbidden, data)
	}
}
