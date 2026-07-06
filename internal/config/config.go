package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"sigs.k8s.io/yaml"
)

const (
	// APIVersion is the supported configuration API version.
	APIVersion = "dev-connect/v1"
	// Kind is the supported configuration document kind.
	Kind = "DevConnectConfig"
	// DefaultFileName is the default dev-connect configuration file name.
	DefaultFileName = "dev-connect.yaml"
)

// Loaded contains a parsed configuration and the source path used.
type Loaded struct {
	Config Config
	Path   string
}

// Loader loads configuration using the approved precedence order.
type Loader struct {
	ExplicitPath   string
	EnvConfig      string
	DefaultDir     string
	DevelopmentDir string
}

// Config is the top-level dev-connect YAML configuration.
type Config struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Contexts   map[string]Context `json:"contexts,omitempty"`
	Clusters   map[string]Cluster `json:"clusters,omitempty"`
	Gateways   map[string]Gateway `json:"gateways,omitempty"`
	Targets    map[string]Target  `json:"targets,omitempty"`
	HostKeys   map[string]string  `json:"hostKeys,omitempty"`
	VSCode     VSCode             `json:"vscode,omitempty"`
}

// Context selects a Kubernetes cluster and gateway.
type Context struct {
	Cluster string `json:"cluster"`
	Gateway string `json:"gateway"`
}

// Cluster describes Kubernetes access settings for a target platform.
type Cluster struct {
	Kubeconfig        string `json:"kubeconfig,omitempty"`
	KubernetesContext string `json:"kubernetesContext,omitempty"`
	Proxy             Proxy  `json:"proxy,omitempty"`
}

// Proxy describes process-scoped proxy overrides for kubectl.
type Proxy struct {
	Enabled    bool   `json:"enabled"`
	HTTPProxy  string `json:"httpProxy,omitempty"`
	HTTPSProxy string `json:"httpsProxy,omitempty"`
	NoProxy    string `json:"noProxy,omitempty"`
}

// Gateway describes a Kubernetes gateway Service.
type Gateway struct {
	Namespace string `json:"namespace"`
	Service   string `json:"service"`
	Port      int    `json:"port"`
}

// Target describes a development server alias.
type Target struct {
	Gateway    string `json:"gateway"`
	User       string `json:"user,omitempty"`
	HostKeyRef string `json:"hostKeyRef"`
}

// VSCode describes VS Code launcher configuration.
type VSCode struct {
	LauncherPath string `json:"launcherPath,omitempty"`
}

// Load reads and validates the first available configuration source.
func (loader Loader) Load() (Loaded, error) {
	candidates, err := loader.candidates()
	if err != nil {
		return Loaded{}, err
	}

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		data, err := os.ReadFile(candidate)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return Loaded{}, fmt.Errorf("read config %q: %w", candidate, err)
		}
		cfg, err := Parse(data)
		if err != nil {
			return Loaded{}, fmt.Errorf("load config %q: %w", candidate, err)
		}
		return Loaded{Config: cfg, Path: candidate}, nil
	}

	return Loaded{}, errors.New("dev-connect config not found")
}

func (loader Loader) candidates() ([]string, error) {
	defaultDir := loader.DefaultDir
	if defaultDir == "" {
		dir, err := DefaultDir()
		if err != nil {
			return nil, err
		}
		defaultDir = dir
	}

	developmentDir := loader.DevelopmentDir
	if developmentDir == "" {
		developmentDir = "."
	}

	return []string{
		loader.ExplicitPath,
		loader.EnvConfig,
		filepath.Join(defaultDir, DefaultFileName),
		filepath.Join(developmentDir, DefaultFileName),
	}, nil
}

// Parse loads and validates a dev-connect YAML configuration.
func Parse(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config YAML: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks that a configuration can be used before external processes start.
func (cfg Config) Validate() error {
	if cfg.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q", cfg.APIVersion)
	}
	if cfg.Kind != Kind {
		return fmt.Errorf("unsupported kind %q", cfg.Kind)
	}
	for name, context := range cfg.Contexts {
		if context.Cluster == "" {
			return fmt.Errorf("context %q missing cluster", name)
		}
		if len(cfg.Clusters) > 0 {
			if _, ok := cfg.Clusters[context.Cluster]; !ok {
				return fmt.Errorf("context %q references unknown cluster %q", name, context.Cluster)
			}
		}
		if context.Gateway == "" {
			return fmt.Errorf("context %q missing gateway", name)
		}
		if len(cfg.Gateways) > 0 {
			if _, ok := cfg.Gateways[context.Gateway]; !ok {
				return fmt.Errorf("context %q references unknown gateway %q", name, context.Gateway)
			}
		}
	}
	for name, target := range cfg.Targets {
		if target.Gateway == "" {
			return fmt.Errorf("target %q missing gateway", name)
		}
		if len(cfg.Gateways) > 0 {
			if _, ok := cfg.Gateways[target.Gateway]; !ok {
				return fmt.Errorf("target %q references unknown gateway %q", name, target.Gateway)
			}
		}
		if target.HostKeyRef == "" {
			return fmt.Errorf("target %q missing hostKeyRef", name)
		}
		if len(cfg.HostKeys) > 0 {
			if cfg.HostKeys[target.HostKeyRef] == "" {
				return fmt.Errorf("target %q references unknown host key %q", name, target.HostKeyRef)
			}
		}
	}
	for name, gateway := range cfg.Gateways {
		if gateway.Namespace == "" {
			return fmt.Errorf("gateway %q missing namespace", name)
		}
		if gateway.Service == "" {
			return fmt.Errorf("gateway %q missing service", name)
		}
		if gateway.Port <= 0 {
			return fmt.Errorf("gateway %q has invalid port %d", name, gateway.Port)
		}
	}
	return nil
}

// DefaultDir returns the native configuration directory for the current OS.
func DefaultDir() (string, error) {
	return DefaultDirFor(runtime.GOOS, os.Getenv("HOME"), os.Getenv("APPDATA"))
}

// DefaultDirFor returns the native configuration directory for a specific OS.
func DefaultDirFor(goos, home, appData string) (string, error) {
	switch goos {
	case "windows":
		if appData == "" {
			return "", errors.New("APPDATA is required on Windows")
		}
		return filepath.Join(appData, "dev-connect"), nil
	case "darwin":
		if home == "" {
			return "", errors.New("HOME is required on macOS")
		}
		return filepath.Join(home, "Library", "Application Support", "dev-connect"), nil
	default:
		if home == "" {
			return "", errors.New("HOME is required")
		}
		return filepath.Join(home, ".config", "dev-connect"), nil
	}
}
