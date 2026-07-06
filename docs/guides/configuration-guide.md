# dev-connect Configuration Guide

Status: Implementation guide

## Configuration Model

`dev-connect` uses YAML for user configuration and JSON for local session state.

Configuration supports:

- multiple Kubernetes contexts,
- multiple clusters,
- multiple gateways,
- multiple targets,
- optional process-scoped proxy overrides,
- VS Code launcher override,
- pinned SSH host key references.

## Configuration Loading

Load precedence:

1. `--config <path>`
2. `DEV_CONNECT_CONFIG`
3. OS-specific user config path
4. repository or working-directory config path for development

Configuration is validated before starting external processes.

Manual validation:

```text
dev-connect config validate
dev-connect --output json config validate
```

## Default Configuration Locations

The client shall determine configuration file locations using native operating system conventions.

Default locations:

| Operating system | Default directory |
| --- | --- |
| Windows | `%APPDATA%\dev-connect\` |
| Linux | `~/.config/dev-connect/` |
| macOS | `~/Library/Application Support/dev-connect/` |

Linux follows the XDG Base Directory Specification.

The effective configuration path shall be discoverable with:

```text
dev-connect config location
```

## Example Shape

```yaml
apiVersion: dev-connect/v1
kind: DevConnectConfig
contexts:
  default:
    cluster: private-cloud-dev
    gateway: dev01
clusters:
  private-cloud-dev:
    kubeconfig: ""
    kubernetesContext: platform-dev
    proxy:
      enabled: false
      httpProxy: ""
      httpsProxy: ""
      noProxy: ""
gateways:
  dev01:
    namespace: dev-connect
    service: dev-connect-gateway-dev01
    port: 22
targets:
  dev01:
    gateway: dev01
    user: developer
    hostKeyRef: dev01
vscode:
  launcherPath: ""
```

## Proxy Overrides

By default, `kubectl` inherits the user's normal environment and enterprise proxy behavior.

If proxy overrides are configured, they apply only to `kubectl` child processes started by `dev-connect`.

Proxy overrides shall not modify:

- operating system proxy settings,
- corporate proxy configuration,
- kubeconfig,
- shell profiles,
- persistent environment variables.

## VS Code Launcher Discovery

Discovery order:

1. `code` in `PATH`.
2. OS-specific default installation paths.
3. User-configured launcher path.

Windows defaults:

- `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd`
- `C:\Program Files\Microsoft VS Code\bin\code.cmd`
- `C:\Program Files (x86)\Microsoft VS Code\bin\code.cmd`

macOS defaults:

- `/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code`
- `/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code`

Linux defaults:

- `/usr/bin/code`
- `/usr/local/bin/code`
- `/snap/bin/code`

## Session State

Session state is stored as JSON.

Session state shall not contain:

- passwords,
- SSH private keys,
- Kubernetes tokens,
- kubeconfig content,
- proxy credentials.

## Schema Generation

JSON schemas shall be generated automatically from the released implementation.

Schemas shall be versioned, for example:

```text
schemas/v1/
```

Changes to existing schemas shall follow semantic versioning. Breaking schema changes require a new major schema version.
