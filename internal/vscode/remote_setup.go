package vscode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const defaultRemoteProxySupport = "override"

// RemoteSetupOptions describes optional VS Code Server setup on a remote SSH target.
type RemoteSetupOptions struct {
	TargetAlias   string
	SSHConfigPath string
	SSHPath       string
	HTTPProxy     string
	HTTPSProxy    string
	NoProxy       string
	ProxySupport  string
	BatchMode     *bool
}

// ConfigureRemote writes VS Code Server proxy setup files on the remote SSH target.
func ConfigureRemote(ctx context.Context, options RemoteSetupOptions) error {
	if options.TargetAlias == "" {
		return errors.New("target alias is required")
	}
	if options.HTTPProxy == "" && options.HTTPSProxy == "" {
		return errors.New("remote VS Code setup requires httpProxy or httpsProxy")
	}

	sshPath := options.SSHPath
	if sshPath == "" {
		sshPath = "ssh"
	}

	cmd := exec.CommandContext(ctx, sshPath, remoteSetupSSHArgs(options)...)
	cmd.Stdin = strings.NewReader(RenderRemoteSetupScript(options))

	var stderr bytes.Buffer
	if useBatchMode(options.BatchMode) {
		cmd.Stderr = &stderr
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("configure remote VS Code server: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func remoteSetupSSHArgs(options RemoteSetupOptions) []string {
	args := []string{}
	if options.SSHConfigPath != "" {
		args = append(args, "-F", options.SSHConfigPath)
	}
	if useBatchMode(options.BatchMode) {
		args = append(args, "-o", "BatchMode=yes")
	}
	return append(args, options.TargetAlias, "bash -s")
}

func useBatchMode(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

// RenderRemoteSetupScript returns a POSIX shell script that prepares VS Code Server proxy settings.
func RenderRemoteSetupScript(options RemoteSetupOptions) string {
	proxySupport := options.ProxySupport
	if proxySupport == "" {
		proxySupport = defaultRemoteProxySupport
	}

	settings := map[string]any{
		"http.proxySupport": proxySupport,
	}
	if proxy := primaryProxy(options); proxy != "" {
		settings["http.proxy"] = proxy
	}
	if noProxy := splitNoProxy(options.NoProxy); len(noProxy) > 0 {
		settings["http.noProxy"] = noProxy
	}
	settingsJSON, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		panic(err)
	}

	serverEnv := []string{"# BEGIN dev-connect remote proxy"}
	if options.HTTPSProxy != "" {
		serverEnv = append(serverEnv,
			"export HTTPS_PROXY="+shellQuote(options.HTTPSProxy),
			"export https_proxy="+shellQuote(options.HTTPSProxy),
		)
	}
	if options.HTTPProxy != "" {
		serverEnv = append(serverEnv,
			"export HTTP_PROXY="+shellQuote(options.HTTPProxy),
			"export http_proxy="+shellQuote(options.HTTPProxy),
		)
	}
	if options.NoProxy != "" {
		serverEnv = append(serverEnv,
			"export NO_PROXY="+shellQuote(options.NoProxy),
			"export no_proxy=\"$NO_PROXY\"",
		)
	}
	serverEnv = append(serverEnv, "# END dev-connect remote proxy")

	return `set -eu
mkdir -p "$HOME/.vscode-server" "$HOME/.vscode-server/data/Machine"

cat > "$HOME/.vscode-server/server-env-setup" <<'DEV_CONNECT_SERVER_ENV'
` + strings.Join(serverEnv, "\n") + `
DEV_CONNECT_SERVER_ENV
chmod 600 "$HOME/.vscode-server/server-env-setup"

cat > "$HOME/.vscode-server/data/Machine/dev-connect-settings.json" <<'DEV_CONNECT_SETTINGS'
` + string(settingsJSON) + `
DEV_CONNECT_SETTINGS

settings="$HOME/.vscode-server/data/Machine/settings.json"
desired="$HOME/.vscode-server/data/Machine/dev-connect-settings.json"
if command -v python3 >/dev/null 2>&1; then
  python3 - "$settings" "$desired" <<'PY'
import json
import sys
from pathlib import Path

settings_path = Path(sys.argv[1])
desired_path = Path(sys.argv[2])
try:
    current = json.loads(settings_path.read_text()) if settings_path.exists() else {}
    if not isinstance(current, dict):
        current = {}
except Exception:
    current = {}
desired = json.loads(desired_path.read_text())
current.update(desired)
settings_path.write_text(json.dumps(current, indent=2) + "\n")
PY
else
  cp "$desired" "$settings"
fi
chmod 600 "$settings"
rm -f "$desired"
`
}

func primaryProxy(options RemoteSetupOptions) string {
	if options.HTTPSProxy != "" {
		return options.HTTPSProxy
	}
	return options.HTTPProxy
}

func splitNoProxy(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
