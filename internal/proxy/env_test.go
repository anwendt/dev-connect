package proxy

import "testing"

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

func valueOf(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			return entry[len(prefix):]
		}
	}
	return ""
}
