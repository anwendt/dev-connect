package vscode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultLauncherPathsMacOS(t *testing.T) {
	got := DefaultLauncherPaths("darwin", "")

	want := "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
	if len(got) == 0 || got[0] != want {
		t.Fatalf("first macOS launcher path = %v, want %q", got, want)
	}
}

func TestResolveLauncherPrefersPath(t *testing.T) {
	tempDir := t.TempDir()
	codePath := filepath.Join(tempDir, "code")
	if err := os.WriteFile(codePath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write fake code: %v", err)
	}

	got, err := ResolveLauncher(Options{
		LookPath: func(name string) (string, error) {
			if name != "code" {
				t.Fatalf("look path name = %q, want code", name)
			}
			return codePath, nil
		},
		Exists: func(path string) bool {
			return path == codePath
		},
		GOOS: "darwin",
	})
	if err != nil {
		t.Fatalf("resolve launcher: %v", err)
	}
	if got != codePath {
		t.Fatalf("launcher = %q, want %q", got, codePath)
	}
}

func TestResolveLauncherUsesConfiguredPathAfterDefaults(t *testing.T) {
	configuredPath := "/custom/code"

	got, err := ResolveLauncher(Options{
		LookPath: func(string) (string, error) {
			return "", os.ErrNotExist
		},
		Exists: func(path string) bool {
			return path == configuredPath
		},
		GOOS:           "darwin",
		ConfiguredPath: configuredPath,
	})
	if err != nil {
		t.Fatalf("resolve launcher: %v", err)
	}
	if got != configuredPath {
		t.Fatalf("launcher = %q, want %q", got, configuredPath)
	}
}
