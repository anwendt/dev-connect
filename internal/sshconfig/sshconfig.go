package sshconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileMode = 0o600

// SessionOptions describes temporary SSH files for one dev-connect session.
type SessionOptions struct {
	Dir       string
	Alias     string
	User      string
	LocalHost string
	LocalPort int
	HostKey   string
}

// SessionFiles identifies generated temporary SSH files.
type SessionFiles struct {
	ConfigPath     string
	KnownHostsPath string
}

// WriteSessionFiles writes temporary SSH config and known_hosts files.
func WriteSessionFiles(options SessionOptions) (SessionFiles, error) {
	if err := validate(options); err != nil {
		return SessionFiles{}, err
	}
	if err := os.MkdirAll(options.Dir, 0o700); err != nil {
		return SessionFiles{}, fmt.Errorf("create SSH temp directory: %w", err)
	}

	files := SessionFiles{
		ConfigPath:     filepath.Join(options.Dir, "ssh_config"),
		KnownHostsPath: filepath.Join(options.Dir, "known_hosts"),
	}

	knownHosts := fmt.Sprintf("[%s]:%d %s\n", options.LocalHost, options.LocalPort, strings.TrimSpace(options.HostKey))
	if err := writeRestricted(files.KnownHostsPath, []byte(knownHosts)); err != nil {
		return SessionFiles{}, err
	}

	config := renderConfig(options, files.KnownHostsPath)
	if err := writeRestricted(files.ConfigPath, []byte(config)); err != nil {
		return SessionFiles{}, err
	}

	return files, nil
}

// Cleanup removes generated temporary SSH files.
func Cleanup(files SessionFiles) error {
	for _, path := range []string{files.ConfigPath, files.KnownHostsPath} {
		if path == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}
	return nil
}

func validate(options SessionOptions) error {
	if options.Dir == "" {
		return errors.New("SSH config directory is required")
	}
	if options.Alias == "" {
		return errors.New("SSH alias is required")
	}
	if options.LocalHost == "" {
		return errors.New("local host is required")
	}
	if options.LocalPort <= 0 {
		return fmt.Errorf("invalid local port %d", options.LocalPort)
	}
	if strings.TrimSpace(options.HostKey) == "" {
		return errors.New("pinned SSH host key is required")
	}
	return nil
}

func renderConfig(options SessionOptions, knownHostsPath string) string {
	lines := []string{
		"Host " + options.Alias,
		"  HostName " + options.LocalHost,
		fmt.Sprintf("  Port %d", options.LocalPort),
		"  UserKnownHostsFile " + knownHostsPath,
		"  StrictHostKeyChecking yes",
		"  IdentitiesOnly yes",
	}
	if options.User != "" {
		lines = append(lines[:3], append([]string{"  User " + options.User}, lines[3:]...)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeRestricted(path string, data []byte) error {
	if err := os.WriteFile(path, data, fileMode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, fileMode); err != nil {
		return fmt.Errorf("set permissions on %s: %w", path, err)
	}
	return nil
}
