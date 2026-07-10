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
    identityFile: /Users/developer/.ssh/dev01
vscode:
  launcherPath: ""
```

Validated examples are available under:

```text
examples/config/
```

The examples cover a minimal single-target setup, process-scoped proxy overrides
for `kubectl`, and a multi-cluster configuration.

## Proxy Overrides

By default, `kubectl` inherits the user's normal environment and enterprise proxy behavior.

If proxy overrides are configured, they apply only to `kubectl` child processes started by `dev-connect`.

Proxy overrides shall not modify:

- operating system proxy settings,
- corporate proxy configuration,
- kubeconfig,
- shell profiles,
- persistent environment variables.

Example:

```yaml
clusters:
  central-dev:
    kubeconfig: /Users/andreswendt/.kube/central-dev-cluster.yaml
    kubernetesContext: ""
    proxy:
      enabled: true
      httpProxy: http://proxy.example.corp:8080
      httpsProxy: http://proxy.example.corp:8080
      noProxy: 127.0.0.1,localhost,.svc,.cluster.local,172.28.0.0/16
```

This override is applied only to the `kubectl` process used for preflight checks
and `kubectl port-forward`. It does not affect VS Code, SSH, the user's shell, or
the operating system proxy configuration.

If the enterprise workstation proxy configuration already works for `kubectl`,
leave the override disabled:

```yaml
proxy:
  enabled: false
```

Do not put proxy credentials into the configuration file. Use the enterprise
proxy mechanism already configured on the workstation or an approved secret
handling mechanism outside `dev-connect`.

## Pinned Host Keys

Every target must have an approved pinned SSH host key. By default,
`dev-connect` uses the target name as the host key reference:

```yaml
targets:
  dev01:
    gateway: dev01
    user: developer
hostKeys:
  dev01: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
```

Set `hostKeyRef` only when the key inventory uses a different name:

```yaml
targets:
  dev01:
    gateway: dev01
    user: developer
    hostKeyRef: ubuntu-dev01-ed25519
hostKeys:
  ubuntu-dev01-ed25519: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExamplePinnedHostKeyReplaceFromGitOpsInventory dev01
```

The host key value must come from the enterprise GitOps host key inventory. The
client writes the key into a temporary `known_hosts` file and keeps
`StrictHostKeyChecking yes`.

## SSH Identity Files

Targets may define a local private key path:

```yaml
targets:
  dev01:
    gateway: dev01
    user: developer
    identityFile: /Users/developer/.ssh/dev01
```

On Windows:

```yaml
targets:
  dev01:
    gateway: dev01
    user: developer
    identityFile: C:\Users\developer\.ssh\dev01
```

The key file remains local. `dev-connect` does not read, copy, log, or store the
private key material. It only writes an `IdentityFile` directive into the
temporary SSH configuration generated for the session.

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
