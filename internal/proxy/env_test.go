package proxy

import (
	"strings"
	"testing"
)

func TestBuildEnvAppliesOverridesWithoutMutatingBase(t *testing.T) {
	base := []string{
		"PATH=/usr/bin",
		"HTTPS_PROXY=http://enterprise.example",
	}

	got := BuildEnv(base, Config{
		Enabled:    true,
		HTTPProxy:  "http://override-http.example",
		HTTPSProxy: "http://override-https.example",
		NoProxy:    "localhost,127.0.0.1",
	})

	if valueOf(got, "HTTPS_PROXY") != "http://override-https.example" {
		t.Fatalf("HTTPS_PROXY = %q", valueOf(got, "HTTPS_PROXY"))
	}
	if valueOf(got, "HTTP_PROXY") != "http://override-http.example" {
		t.Fatalf("HTTP_PROXY = %q", valueOf(got, "HTTP_PROXY"))
	}
	if valueOf(got, "NO_PROXY") != "localhost,127.0.0.1" {
		t.Fatalf("NO_PROXY = %q", valueOf(got, "NO_PROXY"))
	}
	if valueOf(base, "HTTPS_PROXY") != "http://enterprise.example" {
		t.Fatalf("base environment mutated: %#v", base)
	}
}

func TestBuildEnvReturnsCopyWhenDisabled(t *testing.T) {
	base := []string{"PATH=/usr/bin"}
	got := BuildEnv(base, Config{Enabled: false, HTTPSProxy: "http://ignored.example"})

	if len(got) != len(base) || got[0] != base[0] {
		t.Fatalf("env = %#v, want copy of %#v", got, base)
	}
	got[0] = "PATH=/changed"
	if base[0] != "PATH=/usr/bin" {
		t.Fatalf("base environment mutated: %#v", base)
	}
}

func TestBuildEnvReplacesWindowsProxyVariablesCaseInsensitively(t *testing.T) {
	base := []string{
		"Path=C:\\Windows\\System32",
		"Https_Proxy=http://enterprise.example",
		"No_Proxy=localhost",
	}

	got := buildEnvForOS("windows", base, Config{
		Enabled:    true,
		HTTPSProxy: "http://override.example",
		NoProxy:    "localhost,127.0.0.1",
	})

	if valueOfCaseInsensitive(got, "HTTPS_PROXY") != "http://override.example" {
		t.Fatalf("HTTPS_PROXY = %q", valueOfCaseInsensitive(got, "HTTPS_PROXY"))
	}
	if valueOfCaseInsensitive(got, "NO_PROXY") != "localhost,127.0.0.1" {
		t.Fatalf("NO_PROXY = %q", valueOfCaseInsensitive(got, "NO_PROXY"))
	}
	if countKeyCaseInsensitive(got, "HTTPS_PROXY") != 1 {
		t.Fatalf("HTTPS_PROXY entries = %d in %#v", countKeyCaseInsensitive(got, "HTTPS_PROXY"), got)
	}
	if countKeyCaseInsensitive(got, "NO_PROXY") != 1 {
		t.Fatalf("NO_PROXY entries = %d in %#v", countKeyCaseInsensitive(got, "NO_PROXY"), got)
	}
}

func valueOf(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			return entry[len(prefix):]
		}
	}
	return ""
}

func valueOfCaseInsensitive(env []string, key string) string {
	for _, entry := range env {
		name, value, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}

func countKeyCaseInsensitive(env []string, key string) int {
	count := 0
	for _, entry := range env {
		name, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(name, key) {
			count++
		}
	}
	return count
}
