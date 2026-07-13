# dev-connect Config Examples

This directory contains validated example configuration files for `dev-connect`.

All examples use the supported configuration API:

```yaml
apiVersion: dev-connect/v1
kind: DevConnectConfig
```

The examples are intentionally usable as starting points, but pinned SSH host
keys are placeholders. Replace every `hostKeys` value with the approved host key
from the enterprise GitOps host key inventory before using a configuration.

Targets may set `identityFile` to select a local private key without editing the
user's SSH config. The private key file remains local and is not copied or
stored by `dev-connect`.

When `hostKeyRef` is omitted, `dev-connect` uses the target name as the host key
reference. For example, target `dev01` uses `hostKeys.dev01`.

## Examples

| File | Purpose |
| --- | --- |
| `dev01-basic.yaml` | Minimal single-cluster configuration without proxy override. |
| `dev01-proxy.yaml` | Single-cluster configuration with a process-scoped proxy override for `kubectl`. |
| `multi-cluster-proxy.yaml` | Multiple Rancher/Kubernetes contexts, multiple gateways, and per-cluster proxy settings. |
| `windows-bundled-kubectl.yaml` | Windows configuration that points to a bundled `kubectl.exe` next to `dev-connect.exe`. |

## Usage

Validate an example:

```text
dev-connect --config examples/config/dev01-basic.yaml config validate
```

Connect without launching VS Code:

```text
dev-connect --config examples/config/dev01-basic.yaml connect dev01 --no-code
```

Connect with machine-readable output:

```text
dev-connect --config examples/config/dev01-proxy.yaml --output json connect dev01 --no-code
```

Select a named dev-connect context:

```text
dev-connect --config examples/config/multi-cluster-proxy.yaml --context central-dev connect dev01
```

## kubectl Discovery

`dev-connect` delegates all Kubernetes communication to a `kubectl` subprocess.
Windows users do not need a global `kubectl` installation when they use the
Windows bundle release artifact. The client searches for `kubectl.exe` next to
`dev-connect.exe` before falling back to `PATH` and default locations.

You can also set a cluster-specific path:

```yaml
clusters:
  central-dev:
    kubectlPath: C:\Program Files\dev-connect\kubectl.exe
```

## Proxy Behavior

Proxy settings under `clusters.<name>.proxy` are optional.

When `enabled: false`, `kubectl` inherits the normal user environment, including
the enterprise proxy configuration already configured on the workstation.

When `enabled: true`, `dev-connect` applies the configured values only to the
`kubectl` process it starts. It does not modify:

- operating system proxy settings,
- shell profile files,
- kubeconfig,
- corporate proxy configuration,
- global environment variables.

Use a proxy override only when the default enterprise workstation proxy settings
are not sufficient for the selected Kubernetes or Rancher endpoint.

## VS Code Profile Behavior

The examples use an isolated VS Code user-data directory:

```yaml
vscode:
  isolatedUserDataDir: true
```

This allows `dev-connect` to pass the generated temporary SSH config to VS Code
Remote SSH without changing the user's normal VS Code settings. To use the
normal VS Code user profile, set `isolatedUserDataDir: false` and make sure the
target alias is resolvable through the user's normal Remote SSH configuration.

For GitHub Copilot/browser authentication flows that should use the normal VS
Code profile, enable managed user SSH config as well:

```yaml
ssh:
  manageUserConfig: true
  userConfigPath: ""
vscode:
  isolatedUserDataDir: false
```

`dev-connect connect` writes only a marked `dev-connect` block to the user's
OpenSSH config. `dev-connect disconnect` removes that block again.
