package sshconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// UserConfigOptions describes a managed block in the user's SSH config.
type UserConfigOptions struct {
	ConfigPath     string
	Alias          string
	User           string
	IdentityFile   string
	LocalHost      string
	LocalPort      int
	KnownHostsPath string
}

// DefaultUserConfigPath returns the platform default OpenSSH user config path.
func DefaultUserConfigPath() (string, error) {
	return DefaultUserConfigPathFor(runtime.GOOS, os.Getenv("HOME"), os.Getenv("USERPROFILE"))
}

// DefaultUserConfigPathFor returns the platform default OpenSSH user config path.
func DefaultUserConfigPathFor(goos, home, userProfile string) (string, error) {
	base := home
	if goos == "windows" {
		base = userProfile
		if base == "" {
			base = home
		}
	}
	if base == "" {
		return "", errors.New("home directory is required for SSH config path")
	}
	if goos != "windows" {
		return strings.TrimRight(base, "/") + "/.ssh/config", nil
	}
	return filepath.Join(base, ".ssh", "config"), nil
}

// UpsertManagedBlock writes or replaces the dev-connect managed block.
func UpsertManagedBlock(options UserConfigOptions) error {
	if err := validateUserConfigOptions(options); err != nil {
		return err
	}
	content, err := readOptional(options.ConfigPath)
	if err != nil {
		return err
	}
	updated := prependManagedBlock(removeManagedBlock(content, options.Alias), renderManagedBlock(options))
	return writeUserConfig(options.ConfigPath, updated)
}

// RemoveManagedBlock removes the dev-connect managed block for an alias.
func RemoveManagedBlock(configPath, alias string) error {
	if configPath == "" || alias == "" {
		return nil
	}
	content, err := readOptional(configPath)
	if err != nil {
		return err
	}
	updated := removeManagedBlock(content, alias)
	if updated == content {
		return nil
	}
	return writeUserConfig(configPath, updated)
}

func validateUserConfigOptions(options UserConfigOptions) error {
	if options.ConfigPath == "" {
		return errors.New("user SSH config path is required")
	}
	if options.Alias == "" {
		return errors.New("SSH alias is required")
	}
	if strings.ContainsAny(options.Alias, "\r\n") {
		return errors.New("SSH alias must not contain newlines")
	}
	if options.LocalHost == "" {
		return errors.New("local host is required")
	}
	if options.LocalPort <= 0 {
		return fmt.Errorf("invalid local port %d", options.LocalPort)
	}
	if options.KnownHostsPath == "" {
		return errors.New("known_hosts path is required")
	}
	return nil
}

func readOptional(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read user SSH config: %w", err)
	}
	return string(data), nil
}

func renderManagedBlock(options UserConfigOptions) string {
	config := renderConfig(SessionOptions{
		Alias:        options.Alias,
		User:         options.User,
		IdentityFile: options.IdentityFile,
		LocalHost:    options.LocalHost,
		LocalPort:    options.LocalPort,
	}, options.KnownHostsPath)
	return beginMarker(options.Alias) + "\n" + config + endMarker(options.Alias) + "\n"
}

func prependManagedBlock(content, block string) string {
	content = strings.TrimLeft(content, "\r\n")
	if content == "" {
		return block
	}
	return block + "\n" + content
}

func removeManagedBlock(content, alias string) string {
	begin := beginMarker(alias)
	end := endMarker(alias)
	for {
		start := strings.Index(content, begin)
		if start < 0 {
			return strings.TrimLeft(content, "\r\n")
		}
		stop := strings.Index(content[start:], end)
		if stop < 0 {
			return content
		}
		stop += start + len(end)
		for stop < len(content) && (content[stop] == '\r' || content[stop] == '\n') {
			stop++
		}
		content = content[:start] + content[stop:]
	}
}

func writeUserConfig(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create user SSH config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), fileMode); err != nil {
		return fmt.Errorf("write user SSH config: %w", err)
	}
	return os.Chmod(path, fileMode)
}

func beginMarker(alias string) string {
	return "# BEGIN dev-connect " + alias
}

func endMarker(alias string) string {
	return "# END dev-connect " + alias
}
