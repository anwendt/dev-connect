package vscode

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Options controls VS Code launcher resolution.
type Options struct {
	GOOS           string
	LocalAppData   string
	ConfiguredPath string
	LookPath       func(string) (string, error)
	Exists         func(string) bool
}

// ResolveLauncher finds the VS Code launcher without starting VS Code.
func ResolveLauncher(options Options) (string, error) {
	lookPath := options.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	exists := options.Exists
	if exists == nil {
		exists = fileExists
	}

	if path, err := lookPath("code"); err == nil && exists(path) {
		return path, nil
	}

	goos := options.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	for _, path := range DefaultLauncherPaths(goos, options.LocalAppData) {
		if exists(path) {
			return path, nil
		}
	}

	if options.ConfiguredPath != "" && exists(options.ConfiguredPath) {
		return options.ConfiguredPath, nil
	}

	return "", errors.New("VS Code launcher not found")
}

// DefaultLauncherPaths returns documented VS Code launcher paths for an OS.
func DefaultLauncherPaths(goos, localAppData string) []string {
	switch goos {
	case "windows":
		paths := []string{}
		if localAppData != "" {
			paths = append(paths, strings.TrimRight(localAppData, `\/`)+`\Programs\Microsoft VS Code\bin\code.cmd`)
		}
		paths = append(paths,
			`C:\Program Files\Microsoft VS Code\bin\code.cmd`,
			`C:\Program Files (x86)\Microsoft VS Code\bin\code.cmd`,
		)
		return paths
	case "darwin":
		return []string{
			"/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			"/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code",
		}
	default:
		return []string{
			"/usr/bin/code",
			"/usr/local/bin/code",
			"/snap/bin/code",
		}
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
