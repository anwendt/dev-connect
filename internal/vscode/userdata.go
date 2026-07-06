package vscode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFileMode = 0o600

// PrepareUserDataDir writes session-scoped VS Code user settings.
func PrepareUserDataDir(dir, sshConfigPath string) error {
	if dir == "" {
		return errors.New("VS Code user data directory is required")
	}
	if sshConfigPath == "" {
		return errors.New("SSH config path is required")
	}

	userDir := filepath.Join(dir, "User")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		return fmt.Errorf("create VS Code user settings directory: %w", err)
	}

	settings := map[string]string{
		"remote.SSH.configFile": sshConfigPath,
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal VS Code settings: %w", err)
	}
	settingsPath := filepath.Join(userDir, "settings.json")
	if err := os.WriteFile(settingsPath, append(data, '\n'), settingsFileMode); err != nil {
		return fmt.Errorf("write VS Code settings: %w", err)
	}
	return os.Chmod(settingsPath, settingsFileMode)
}

// CleanupUserDataDir removes the generated session-scoped VS Code user data directory.
func CleanupUserDataDir(dir string) error {
	if dir == "" {
		return nil
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove VS Code user data directory: %w", err)
	}
	return nil
}
