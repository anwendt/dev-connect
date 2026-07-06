package proxy

import "strings"

// Config describes process-scoped proxy overrides.
type Config struct {
	Enabled    bool
	HTTPProxy  string
	HTTPSProxy string
	NoProxy    string
}

// BuildEnv returns a process environment with optional proxy overrides.
func BuildEnv(base []string, config Config) []string {
	env := append([]string(nil), base...)
	if !config.Enabled {
		return env
	}

	if config.HTTPProxy != "" {
		env = set(env, "HTTP_PROXY", config.HTTPProxy)
		env = set(env, "http_proxy", config.HTTPProxy)
	}
	if config.HTTPSProxy != "" {
		env = set(env, "HTTPS_PROXY", config.HTTPSProxy)
		env = set(env, "https_proxy", config.HTTPSProxy)
	}
	if config.NoProxy != "" {
		env = set(env, "NO_PROXY", config.NoProxy)
		env = set(env, "no_proxy", config.NoProxy)
	}
	return env
}

func set(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}
