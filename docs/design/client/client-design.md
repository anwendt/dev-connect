# dev-connect Client Design

Status: Approved

## 1. Purpose

This document defines the design for the `dev-connect` client.

Phase 5 targets a cross-platform Go CLI for Windows, Linux, and macOS. This document is a pre-implementation design and does not create Go code, tests, binaries, CI/CD workflows, Kubernetes manifests, Helm charts, or Kustomize files.

## 2. Scope

In scope:

- Go CLI architecture.
- Commands and flags.
- Configuration loading and validation.
- `kubectl` detection and invocation.
- Kubernetes preflight through `kubectl`.
- User-specific proxy overrides scoped to managed `kubectl` processes.
- Free local port allocation.
- Port-forward lifecycle management.
- Temporary SSH configuration and pinned host key handling.
- VS Code launcher discovery and invocation.
- Session state, locking, status, disconnect, reconnect, and cleanup.
- Logging and machine-readable output.
- Cross-platform behavior for Windows, Linux, and macOS.

Out of scope:

- Go implementation files.
- Unit tests and integration tests.
- Gateway manifests.
- Helm and Kustomize implementation.
- CI/CD workflows.

## 3. Design Principles

- `dev-connect` shall never establish direct network connections to Rancher or the Kubernetes API.
- All Kubernetes communication shall be delegated to the locally installed `kubectl` binary.
- `kubectl` shall use the user's existing kubeconfig and enterprise proxy configuration by default.
- Optional user-specific proxy overrides shall be applied only to `kubectl` processes started by `dev-connect`.
- Proxy overrides shall not modify global operating system, corporate, shell, kubeconfig, or persistent environment settings.
- The gateway remains TCP-only and does not perform SSH authentication.
- SSH authentication remains on the target Linux development server.
- The client shall not store SSH private keys, Kubernetes tokens, passwords, or user databases.
- SSH host key pinning is mandatory.

## 4. Commands

CLI framework:

- The client shall use Cobra.
- Cobra is selected because it is Apache License 2.0, common in Kubernetes-related CLIs, mature, supports command hierarchies, and supports shell completion.

The CLI shall support:

```text
dev-connect connect <target>
dev-connect disconnect
dev-connect status
dev-connect list
dev-connect version
dev-connect help
```

Global flags:

```text
--config <path>
--context <name>
--cluster <name>
--gateway <name>
--log-level <error|warn|info|debug>
--log-format <text|json>
--output <text|json>
--no-code
--no-reconnect
--timeout <duration>
```

Output model:

- `--output` controls user-visible command responses.
- `--log-format` controls internal log formatting.
- `--output json` and `--log-format json` are independent.

## 5. Proposed Go Package Boundaries

Recommended package layout for later implementation:

```text
cmd/dev-connect/
internal/config/
internal/kubectl/
internal/proxy/
internal/port/
internal/tunnel/
internal/sshconfig/
internal/vscode/
internal/session/
internal/logging/
internal/output/
internal/platform/
```

Package responsibilities:

- `cli`: command parsing and command orchestration.
- `config`: YAML loading, validation, defaults, and schema versioning.
- `kubectl`: `kubectl` path resolution, command construction, process execution, and output parsing.
- `proxy`: process-scoped proxy environment construction.
- `port`: loopback port allocation.
- `tunnel`: managed `kubectl port-forward` lifecycle.
- `sshconfig`: temporary SSH config and temporary `known_hosts` generation.
- `sshconfig`: optional managed user OpenSSH config block lifecycle.
- `vscode`: VS Code launcher discovery and Remote SSH invocation.
- `session`: state file, lock file, PID validation, stale cleanup.
- `logging`: redacted structured logs.
- `output`: text and JSON command responses.
- `platform`: OS-specific paths, process behavior, file permissions, and launcher defaults.

## 6. Configuration Design

Configuration format:

- YAML.
- YAML parsing shall use `sigs.k8s.io/yaml`.
- Explicit `apiVersion` and `kind`.
- Supports multiple contexts, clusters, gateways, and targets.

Required behavior:

- Load precedence:
  1. `--config <path>`
  2. `DEV_CONNECT_CONFIG`
  3. OS-specific user config path
  4. repository or working-directory config path for development
- Validate before starting any external process.
- Reject unsupported schema versions.
- Reject targets without gateway mapping.
- Reject missing host key references where host key pinning is required.

Proxy configuration:

```yaml
clusters:
  private-cloud-dev:
    kubeconfig: ""
    kubernetesContext: platform-dev
    proxy:
      enabled: false
      httpProxy: ""
      httpsProxy: ""
      noProxy: ""
```

Proxy requirements:

- If `proxy.enabled` is false, inherit the user's normal environment and enterprise proxy behavior.
- If enabled, set proxy variables only in the environment of `kubectl` child processes.
- Do not persist proxy overrides globally.
- Do not modify kubeconfig.

## 7. kubectl Design

Detection:

- Search `PATH`.
- Support explicit configured path.
- Support environment override where documented.
- Run `kubectl version --client` or equivalent.

Invocation:

- All Kubernetes discovery, preflight, and port-forward actions are executed through `kubectl`.
- The client shall not import or use Kubernetes Go client libraries for API calls.
- The client shall not call Rancher APIs.

Required operations:

- Current context validation.
- Namespace access validation.
- Gateway Service discovery.
- Endpoint or Pod readiness discovery where permitted.
- RBAC preflight using `kubectl auth can-i`.
- Functional validation by establishing a temporary `kubectl port-forward`.
- `kubectl port-forward service/<gateway-service> <localPort>:22`.

RBAC validation:

- The connection shall proceed only when both `kubectl auth can-i` and temporary `kubectl port-forward` validation succeed.
- The functional validation is required to detect RBAC restrictions, admission controller restrictions, and Rancher-specific authorization behavior that may not be visible from RBAC checks alone.

Environment:

- Use selected kubeconfig and context.
- Apply optional proxy override only to the process environment.
- Capture stdout/stderr for readiness and diagnostics.

## 8. Port Allocation

Requirements:

- Allocate loopback-only local TCP ports.
- Avoid privileged ports.
- Detect port conflicts.
- Retry bounded allocation attempts.
- Record selected port in session state.

Default:

- IPv4 loopback `127.0.0.1`.
- Future `::1` support may be added after platform verification.

## 9. Tunnel Lifecycle

Startup:

1. Validate config.
2. Detect `kubectl`.
3. Run Kubernetes preflight through `kubectl`.
4. Discover Gateway Service.
5. Allocate local port.
6. Start managed `kubectl port-forward`.
7. Wait for readiness.
8. Generate SSH config and `known_hosts`.
9. Launch VS Code unless `--no-code`.
10. Persist session state.
11. Monitor process.

Readiness:

- `kubectl` reports forwarding on the selected port.
- Local TCP readiness check succeeds.
- Startup timeout is enforced.

Shutdown:

- Stop managed process.
- Remove temporary SSH config.
- Remove temporary `known_hosts`.
- Release lock.
- Remove session state.

## 10. SSH Config and Host Key Pinning

Requirements:

- Generate temporary SSH config per session.
- Generate temporary `known_hosts` from centrally managed pinned host keys.
- Use `StrictHostKeyChecking yes`.
- Use `UserKnownHostsFile <temporary-known-hosts>`.
- Do not write private keys.
- Do not modify user-managed SSH config by default.
- Remove temporary files during cleanup.

Example:

```sshconfig
Host dev01.dev-connect
  HostName 127.0.0.1
  Port 49152
  User developer
  HostKeyAlias dev01
  StrictHostKeyChecking yes
  UserKnownHostsFile /tmp/dev-connect/session/known_hosts
```

## 11. VS Code Launcher Design

Discovery order:

1. `code` executable found in the user's `PATH`.
2. Operating system default installation paths.
3. User-configured launcher path in `dev-connect.yaml`.

Default Windows paths:

- `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd`
- `C:\Program Files\Microsoft VS Code\bin\code.cmd`
- `C:\Program Files (x86)\Microsoft VS Code\bin\code.cmd`

Default macOS paths:

- `/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code`
- `/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code`

Default Linux paths:

- `/usr/bin/code`
- `/usr/local/bin/code`
- `/snap/bin/code`

Behavior:

- Launch VS Code Desktop Remote SSH target.
- Use a session-scoped VS Code profile by default so Remote SSH can read the generated temporary SSH config.
- Support `vscode.isolatedUserDataDir: false` for using the normal VS Code user profile when the target alias is already resolvable there.
- Support `ssh.manageUserConfig: true` for normal VS Code profile mode by adding and later removing a marked user OpenSSH config block.
- Do not launch browser-based VS Code.
- Do not install extensions.
- Do not manage GitHub Copilot.
- `--no-code` starts tunnel only.

Expected command:

```text
code --remote ssh-remote+dev01.dev-connect
```

## 12. Session State and Locking

Session state shall be stored as JSON. Configuration files shall use YAML.

State shall include:

- session ID,
- target alias,
- generated SSH host alias,
- local port,
- kube context,
- namespace,
- gateway service,
- `kubectl` process ID,
- reconnect policy,
- timeout policy,
- temporary file paths,
- timestamps.

State shall not include:

- SSH private keys,
- passwords,
- Kubernetes tokens,
- kubeconfig content.

First release:

- One active managed session per user profile.
- One active port-forward per developer by default.

Locking:

- Prevent concurrent `connect` commands from corrupting session state.
- Allow idempotent `disconnect`.
- Detect stale lock/state safely.

## 13. Reconnect and Timeouts

Reconnect:

- Enabled by default.
- Bounded exponential backoff.
- Preserve target and session metadata.
- Reuse same local port when safe.
- Do not retry SSH credentials.
- Stop after configured max attempts.

Timeouts:

- Idle timeout default: 60 minutes.
- Maximum session duration default: 12 hours.
- Cleanup temporary resources after timeout.
- Do not terminate unrelated user processes.

## 14. Logging and Output

Logging:

- Logging shall use the Go standard library `log/slog` package.
- Metadata only.
- Redact credentials, tokens, private keys, kubeconfig contents, SSH payloads, terminal contents, and file contents.
- Include session ID, target, gateway, namespace, context, local port, start/end time, duration, exit code.

Output:

- Text output by default.
- JSON output with `--output json`.
- JSON output shall be treated as a public and versioned API.
- Every JSON response shall contain an `apiVersion` field.
- Initial JSON response API version shall be `v1`.

Example:

```json
{
  "apiVersion": "v1",
  "status": "Connected",
  "server": "dev01"
}
```

## 15. Dependency Policy

The project shall minimize external dependencies.

Where equivalent functionality exists within the Go standard library or Kubernetes SIG ecosystem, those implementations shall be preferred.

Approved first-implementation dependencies:

- CLI framework: Cobra.
- YAML: `sigs.k8s.io/yaml`.
- Logging: Go standard library `log/slog`.

Mandatory runtime dependencies shall use one of these licenses:

- Apache License 2.0.
- MIT License.
- BSD-2-Clause.
- BSD-3-Clause.

This license policy ensures compatibility with the Apache License 2.0 used by the project.

## 16. Cross-Platform Requirements

Windows:

- Use `%APPDATA%\dev-connect\config.yaml`.
- Handle `.exe` discovery for `kubectl` and VS Code launcher.
- Use Windows-safe process termination.
- Use Windows ACL-compatible temporary file permissions where possible.

macOS:

- Use `~/Library/Application Support/dev-connect/config.yaml`.
- Check common VS Code launcher paths.
- Use POSIX file permissions for temporary files.

Linux:

- Use `~/.config/dev-connect/config.yaml`.
- Check common `code` launcher paths.
- Use POSIX file permissions for temporary files.

## 17. Error and Exit Code Design

The client shall preserve the Phase 2 exit code model.

Errors shall distinguish:

- invalid command,
- configuration failure,
- missing dependency,
- Kubernetes connectivity failure through `kubectl`,
- Kubernetes authorization failure through `kubectl`,
- gateway unavailable,
- port-forward failure,
- VS Code launch failure,
- SSH/VS Code authentication failure,
- cleanup failure,
- reconnect exhausted.

## 18. Testability Requirements for Phase 7

The design shall allow mocks for:

- `kubectl` executable behavior,
- `kubectl port-forward` process,
- filesystem,
- local port allocator,
- VS Code launcher,
- session state,
- clock/timers,
- platform-specific paths.

No implementation tests are created in Phase 5 until this client design is reviewed and approved.

## 19. Phase Gate

This document completes the approved Phase 5 client design. Phase 6 may start.
