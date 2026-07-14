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
- optional `kubectl` executable override,
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
    kubectlPath: ""
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
ssh:
  manageUserConfig: false
  userConfigPath: ""
vscode:
  launcherPath: ""
  isolatedUserDataDir: true
```

Validated examples are available under:

```text
examples/config/
```

The examples cover a minimal single-target setup, process-scoped proxy overrides
for `kubectl`, and a multi-cluster configuration.

## VS Code Profile Mode

`vscode.isolatedUserDataDir` controls whether `dev-connect` launches VS Code
with a session-specific user-data directory.

Default:

```yaml
vscode:
  isolatedUserDataDir: true
```

With the default setting, `dev-connect` creates a temporary VS Code user-data
directory and writes `remote.SSH.configFile` to point at the generated temporary
SSH configuration. This ensures VS Code Remote SSH can resolve the generated
target alias without changing the user's normal VS Code settings.

Normal local VS Code profile mode:

```yaml
vscode:
  isolatedUserDataDir: false
```

With `isolatedUserDataDir: false`, VS Code uses the normal local user profile.
Existing GitHub sign-in state, GitHub Copilot authentication, settings, and
locally installed extensions remain available. In this mode, VS Code Remote SSH
must be able to resolve the target alias through the user's normal Remote SSH
configuration because `dev-connect` does not modify the user's global VS Code
settings.

## User SSH Config Management

`ssh.manageUserConfig` controls whether `dev-connect` writes a managed block to
the user's normal OpenSSH config during `connect` and removes it during
`disconnect`.

Default:

```yaml
ssh:
  manageUserConfig: false
  userConfigPath: ""
```

When enabled, `dev-connect` writes a block like this:

```sshconfig
# BEGIN dev-connect dev01
Host dev01
  HostName 127.0.0.1
  Port 55221
  User developer
  IdentityFile "/Users/developer/.ssh/dev01"
  UserKnownHostsFile "/path/to/dev-connect/session/ssh/known_hosts"
  StrictHostKeyChecking yes
  IdentitiesOnly yes
# END dev-connect dev01
```

Only the block between the `BEGIN` and `END` markers is managed. Existing user
SSH configuration outside that block is preserved.

Default config paths:

- Windows: `%USERPROFILE%\.ssh\config`
- Linux: `~/.ssh/config`
- macOS: `~/.ssh/config`

`userConfigPath` may be set to override the default path.

For GitHub Copilot and browser authentication flows that should use the normal
VS Code profile, use:

```yaml
ssh:
  manageUserConfig: true
  userConfigPath: ""
vscode:
  isolatedUserDataDir: false
```

With this combination, VS Code uses the normal local profile while OpenSSH can
resolve the dev-connect target alias through the managed SSH config block.

## VS Code Remote Server Setup

`vscode.remoteSetup` controls optional VS Code Server configuration on the
remote development host. This is useful when Remote SSH extensions such as
GitHub Copilot must use a proxy that is reachable from the remote host, not the
developer workstation.

Example:

```yaml
vscode:
  isolatedUserDataDir: false
  remoteSetup:
    enabled: true
    sshPath: 'C:\Users\developer\AppData\Local\Programs\Git\usr\bin\ssh.exe'
    httpProxy: http://172.28.236.246:8080
    httpsProxy: http://172.28.236.246:8080
    noProxy: localhost,127.0.0.1,0.0.0.0,10.0.0.0/8,.svc,.cluster.local
    proxySupport: override
    batchMode: true
```

When enabled, `dev-connect connect` uses SSH after the local tunnel is ready and
writes these remote VS Code Server files:

- `~/.vscode-server/server-env-setup`
- `~/.vscode-server/data/Machine/settings.json`

The remote machine settings are merged when `python3` is available on the remote
host. `http.noProxy` is written as a JSON string array because VS Code expects
string values there. `proxySupport` defaults to `override` when omitted.

`sshPath` is optional. Set it when VS Code Remote SSH must use a specific SSH
client, for example Git for Windows SSH with a running Git SSH agent. By
default, the remote setup command runs with `BatchMode=yes`, so the required SSH
key must already be available through the selected SSH client and agent. Set
`batchMode: false` only when the selected SSH client must prompt interactively
for a password or key passphrase during remote setup.

## kubectl Discovery

`dev-connect` never opens direct network connections to Rancher or the
Kubernetes API. All Kubernetes communication is still delegated to a `kubectl`
subprocess.

The client resolves `kubectl` in this order:

1. `--kubectl-path <path>`
2. `DEV_CONNECT_KUBECTL_PATH`
3. `clusters.<name>.kubectlPath`
4. `kubectl` located next to the `dev-connect` executable
5. `kubectl` found in `PATH`
6. documented operating system default locations

Windows clients can use the release bundle that contains both
`dev-connect.exe` and `kubectl.exe` in the same directory. In that case no
separate `kubectl` installation and no `PATH` update is required.

Example:

```yaml
clusters:
  central-dev:
    kubeconfig: 'C:\Users\developer\.kube\central-dev-cluster.yaml'
    kubernetesContext: ""
    kubectlPath: 'C:\Program Files\dev-connect\kubectl.exe'
```

Use single quotes for Windows paths in YAML. Do not use double quotes unless
backslashes are escaped, because YAML treats sequences such as `\U` as escape
syntax.

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
