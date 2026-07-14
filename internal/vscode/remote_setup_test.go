package vscode

import (
	"strings"
	"testing"
)

func TestRenderRemoteSetupScriptWritesProxyEnvironment(t *testing.T) {
	script := RenderRemoteSetupScript(RemoteSetupOptions{
		HTTPProxy:    "http://proxy.example.corp:8080",
		HTTPSProxy:   "http://secure-proxy.example.corp:8080",
		NoProxy:      "localhost, 127.0.0.1,.svc",
		ProxySupport: "override",
	})

	for _, want := range []string{
		`export HTTP_PROXY='http://proxy.example.corp:8080'`,
		`export HTTPS_PROXY='http://secure-proxy.example.corp:8080'`,
		`"http.proxy": "http://secure-proxy.example.corp:8080"`,
		`"http.proxySupport": "override"`,
		`"localhost"`,
		`"127.0.0.1"`,
		`".svc"`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script does not contain %q:\n%s", want, script)
		}
	}
}

func TestRenderRemoteSetupScriptDefaultsProxySupport(t *testing.T) {
	script := RenderRemoteSetupScript(RemoteSetupOptions{HTTPProxy: "http://proxy.example.corp:8080"})

	if !strings.Contains(script, `"http.proxySupport": "override"`) {
		t.Fatalf("script did not default proxy support to override:\n%s", script)
	}
}

func TestRenderRemoteSetupScriptEscapesShellValues(t *testing.T) {
	script := RenderRemoteSetupScript(RemoteSetupOptions{HTTPProxy: "http://proxy.example.corp:8080/a'b"})

	if !strings.Contains(script, `export HTTP_PROXY='http://proxy.example.corp:8080/a'\''b'`) {
		t.Fatalf("script did not shell-escape proxy value:\n%s", script)
	}
}
