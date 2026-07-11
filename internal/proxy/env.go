package proxy

import (
	"runtime"
	"strings"
)

// Config describes process-scoped proxy overrides.
type Config struct {
	Enabled    bool
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

// BuildEnv returns a process environment with optional proxy overrides.
func BuildEnv(base []string, config Config) []string {
	return buildEnvForOS(runtime.GOOS, base, config)
}

func buildEnvForOS(goos string, base []string, config Config) []string {
	env := append([]string(nil), base...)
	if !config.Enabled {
		return env
	}

	if config.HTTPProxy != "" {
		env = set(goos, env, "HTTP_PROXY", config.HTTPProxy)
		env = set(goos, env, "http_proxy", config.HTTPProxy)
	}
	if config.HTTPSProxy != "" {
		env = set(goos, env, "HTTPS_PROXY", config.HTTPSProxy)
		env = set(goos, env, "https_proxy", config.HTTPSProxy)
	}
	if config.NoProxy != "" {
		env = set(goos, env, "NO_PROXY", config.NoProxy)
		env = set(goos, env, "no_proxy", config.NoProxy)
	}
	return env
}

func set(goos string, env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if hasEnvKey(goos, entry, key) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func hasEnvKey(goos, entry, key string) bool {
	name, _, ok := strings.Cut(entry, "=")
	if !ok {
		return false
	}
	if goos == "windows" {
		return strings.EqualFold(name, key)
	}
	return name == key
}
