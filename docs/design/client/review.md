# dev-connect Phase 5 Review

Status: Approved

## Scope

Phase 5 currently covers client design only. It intentionally does not include Go implementation files, tests, binaries, CI/CD workflows, Kubernetes manifests, Helm charts, or Kustomize files.

Primary artifact:

- `client-design.md`

## Review Checklist

Command behavior:

- [x] Commands `connect`, `disconnect`, `status`, `list`, `version`, and `help` are represented.
- [x] Global flags are sufficient.
- [x] `--output json` is separate from `--log-format`.
- [x] Exit code categories are preserved.

kubectl and proxy:

- [x] All Kubernetes communication is delegated to local `kubectl`.
- [x] Direct Rancher and Kubernetes API clients are forbidden.
- [x] Existing kubeconfig behavior is preserved.
- [x] Enterprise proxy behavior is preserved by default.
- [x] Optional proxy overrides are scoped only to managed `kubectl` processes.

Client internals:

- [x] Package boundaries are acceptable.
- [x] Configuration loading and validation are sufficient.
- [x] Port allocation design is sufficient.
- [x] Port-forward lifecycle design is sufficient.
- [x] Session state and locking design are sufficient.
- [x] Reconnect and timeout behavior are sufficient.

Security:

- [x] SSH host key pinning behavior is sufficient.
- [x] Temporary SSH config behavior is sufficient.
- [x] Temporary `known_hosts` behavior is sufficient.
- [x] No credentials are stored in client state.
- [x] Logs and JSON output avoid leaking secrets.

VS Code:

- [x] Launcher discovery order is acceptable.
- [x] Remote SSH launch behavior is acceptable.
- [x] `--no-code` tunnel-only behavior is acceptable.

Cross-platform:

- [x] Windows behavior is sufficient.
- [x] Linux behavior is sufficient.
- [x] macOS behavior is sufficient.
- [x] Temporary file permissions are addressed.
- [x] Process termination behavior is addressed.

Testability:

- [x] External process behavior can be mocked.
- [x] Filesystem behavior can be mocked.
- [x] Time and reconnect behavior can be tested.
- [x] Platform-specific behavior can be tested.

## Required Decisions Before Go Implementation

The following decisions are approved for the first implementation and shall be used as the baseline for Go code and tests:

1. CLI framework:
   - use Cobra.
2. YAML library:
   - use `sigs.k8s.io/yaml`.
3. Logging framework:
   - use Go standard library `log/slog`.
4. JSON output schema:
   - treat JSON output as a public and versioned API.
   - every JSON response contains `apiVersion`.
   - initial API version is `v1`.
5. Session state format:
   - store session state as JSON.
   - keep configuration files as YAML.
6. VS Code launcher discovery:
   - first resolve `code` from the user's `PATH`.
   - then check OS-specific default installation paths.
   - then use user-configured launcher path in `config.yaml`.
   - Windows defaults:
     - `%LOCALAPPDATA%\Programs\Microsoft VS Code\bin\code.cmd`
     - `C:\Program Files\Microsoft VS Code\bin\code.cmd`
     - `C:\Program Files (x86)\Microsoft VS Code\bin\code.cmd`
   - macOS defaults:
     - `/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code`
     - `/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code`
   - Linux defaults:
     - `/usr/bin/code`
     - `/usr/local/bin/code`
     - `/snap/bin/code`
7. Kubernetes RBAC preflight:
   - execute `kubectl auth can-i`.
   - perform functional validation by establishing a temporary `kubectl port-forward`.
   - proceed only if both checks succeed.
8. Dependency policy:
   - minimize external dependencies.
   - prefer Go standard library or Kubernetes SIG ecosystem where equivalent functionality exists.
   - mandatory runtime dependencies must use Apache License 2.0, MIT License, BSD-2-Clause, or BSD-3-Clause.

Remaining decisions before Go implementation:

No Phase 5 client design decisions remain open.

## Approval Gate

Phase 5 is approved. Phase 6 may start.
