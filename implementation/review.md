# dev-connect Implementation Kickoff Review

Status: VS Code Remote SSH launcher foundation completed

## Scope

This phase starts implementation planning after approval of Phases 1 through 10.

This review intentionally does not approve broad production implementation without tests. Each implementation slice shall follow TDD and shall include tests or validation checks before production code.

## Review Checklist

Process:

- [x] Approved architecture is the implementation source.
- [x] Approved functional specification is the implementation source.
- [x] Approved non-functional requirements are the implementation source.
- [x] Approved Kubernetes design is the implementation source.
- [x] Approved client design is the implementation source.
- [x] Approved security specification is the implementation source.
- [x] Approved testing strategy is the implementation source.
- [x] Approved repository structure is the implementation source.
- [x] Approved CI/CD design is the implementation source.
- [x] Approved documentation is the implementation source.

TDD:

- [x] Implementation slices start with tests or validation checks.
- [x] Security and architecture compliance tests are mandatory.
- [x] Makefile targets are part of the implementation foundation.
- [x] External dependencies remain minimized.

Architecture constraints:

- [x] No direct Kubernetes or Rancher API client is allowed.
- [x] Kubernetes access remains delegated to `kubectl`.
- [x] Gateway remains TCP-only.
- [x] SSH authentication remains on the target server.
- [x] SSH host key pinning remains mandatory.
- [x] No public endpoint is introduced.
- [x] No credentials are stored in Kubernetes.

## Approved Decisions Before First Code Slice

The following decisions are approved for the initial Go client setup:

1. Go version:
   - use the currently supported stable Go release for CI and local development.
   - initial baseline: Go 1.26.x.
   - Go does not have a classic LTS model; the project shall track supported Go releases and update regularly.
2. Go module path:
   - use `github.com/anwendt/dev-connect`.
   - the module path shall reflect the future standalone repository, not the current monorepo subdirectory.
3. Initial package import root:
   - use the same root as the module path.
   - implementation packages shall live under `internal/` unless a public package is explicitly approved.
4. Initial package layout:
   - `cmd/dev-connect`
   - `internal/config`
   - `internal/kubectl`
   - `internal/proxy`
   - `internal/session`
   - `internal/sshconfig`
   - `internal/vscode`
   - `internal/logging`
   - `internal/output`
5. First implementation commit:
   - include tooling plus passing foundation tests.
   - do not establish a permanently red initial baseline.
   - continue TDD by adding failing feature tests before each new feature slice.
6. First local test target:
   - macOS.
   - OS-specific logic shall be isolated behind interfaces and remain cross-platform from the beginning.
   - follow-up target priority: Windows, then Linux.

Remaining decisions before first code slice:

No initial Go client setup decisions remain open.

## Approval Gate

The first implementation slice is complete.

## Slice 1 Result

Implemented foundation artifacts:

- Go module using `github.com/anwendt/dev-connect`.
- Makefile with foundation targets.
- `.golangci.yml`.
- Cobra root command with `version` and `config location`.
- YAML configuration parsing and validation foundation.
- JSON output response foundation with `apiVersion`.
- macOS VS Code launcher path detection test.
- fake `kubectl` harness skeleton.
- forbidden Kubernetes/Rancher client import test.

Verification:

- `make fmt-check`: passed.
- `make test-unit`: passed.
- `make test-security`: passed.
- `make test`: passed.
- `make build`: passed.
- `make clean`: completed and removed generated binary artifacts.
- `make lint`: passed with `golangci-lint` 2.12.2.

Notes:

- Go emitted macOS `xcrun_db` cache warnings from the sandboxed environment, but the relevant Make targets completed successfully.
- Makefile targets use project-local caches under `.cache/` for sandbox-safe local execution.
- `go mod tidy` required network access to download and verify approved dependencies.

## Slice 2 Result

Implemented CLI skeleton artifacts:

- Approved command surface:
  - `connect <target>`
  - `disconnect`
  - `status`
  - `list`
  - `version`
  - `help`
  - `config location`
- Approved global flags:
  - `--config`
  - `--context`
  - `--cluster`
  - `--gateway`
  - `--log-level`
  - `--log-format`
  - `--output`
  - `--no-code`
  - `--no-reconnect`
  - `--timeout`
- JSON output remains versioned with `apiVersion`.
- `--output` and `--log-format` are validated independently.
- Skeleton commands do not start `kubectl`, SSH, or VS Code.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

## Slice 3 Result

Implemented configuration loading artifacts:

- YAML configuration loader using `sigs.k8s.io/yaml`.
- Approved configuration load precedence:
  - explicit `--config` path,
  - `DEV_CONNECT_CONFIG`,
  - OS-specific default directory,
  - development working-directory fallback.
- Default file name: `dev-connect.yaml`.
- Native default configuration directories:
  - Windows: `%APPDATA%\dev-connect`
  - Linux: `~/.config/dev-connect`
  - macOS: `~/Library/Application Support/dev-connect`
- Config validation before `connect` side effects.
- Gateway reference validation for targets.
- Gateway namespace, Service, and port validation.
- Proxy override parsing without environment mutation.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

## Slice 4 Result

Implemented external process abstraction artifacts:

- `kubectl` command model.
- `kubectl auth can-i` command builder.
- `kubectl port-forward service/<name>` command builder.
- fake `kubectl` runner for deterministic tests.
- `kubectl` exit error model.
- process-scoped proxy environment builder.
- proxy override handling without mutating the base environment.
- VS Code launcher resolver.
- VS Code discovery priority:
  - `code` from `PATH`,
  - OS default paths,
  - configured launcher path.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

Notes:

- Slice 4 does not execute real `kubectl`, real VS Code, SSH, or external network calls.

## Slice 5 Result

Implemented SSH config and host key pinning artifacts:

- Temporary SSH config writer.
- Temporary `known_hosts` writer.
- Pinned host key entry for `[127.0.0.1]:<localPort>`.
- `StrictHostKeyChecking yes`.
- `UserKnownHostsFile` bound to the generated temporary `known_hosts`.
- No generated `IdentityFile`.
- Missing host key rejection.
- Host key rotation overwrites obsolete pinned key material.
- Cleanup removes session-scoped SSH config and `known_hosts`.
- Generated files use restrictive `0600` permissions.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

Notes:

- Slice 5 does not perform a real SSH connection.
- The gateway still does not store or process SSH private keys.

## Slice 6 Result

Implemented tunnel lifecycle foundation artifacts:

- Loopback-only local TCP port allocation abstraction.
- Sandbox-safe port allocation tests through an injected port probe.
- `kubectl port-forward` supervisor using injected `kubectl.Runner`.
- Port-forward readiness detection from `kubectl` output.
- Automatic reconnect retry when enabled.
- No reconnect retry when disabled.
- JSON session state store.
- Session state credential rejection.
- Session state cleanup.
- Exclusive lock file for one active managed session.
- Stale state detection through injected process-existence check.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

Notes:

- Slice 6 does not start a real `kubectl port-forward`.
- Unit tests do not require real network listeners because the sandbox blocks loopback bind operations.

## Slice 7 Result

Implemented Kubernetes gateway artifacts:

- Kustomize base:
  - Namespace.
  - ServiceAccount with `automountServiceAccountToken: false`.
  - HAProxy ConfigMap.
  - Deployment.
  - ClusterIP Service.
  - NetworkPolicy.
  - PodDisruptionBudget.
  - least-privilege Role and RoleBinding.
- Kustomize autoscaling overlay:
  - optional HorizontalPodAutoscaler.
  - HPA is not included in the base by default.
- Helm chart:
  - Chart metadata.
  - values file.
  - ServiceAccount template.
  - HAProxy ConfigMap template.
  - Deployment template.
  - ClusterIP Service template.
  - optional HPA template gated by `autoscaling.enabled`.
  - PDB template.
  - RBAC template.
  - NetworkPolicy template.
- Makefile targets:
  - `make helm-lint`
  - `make helm-template`
  - `make kustomize-build`

Security and design assertions:

- no Ingress.
- no `LoadBalancer`.
- no `NodePort`.
- Service port `22` targets container port `2222`.
- HAProxy uses TCP mode and binds only to non-privileged port `2222`.
- Deployment runs non-root and drops Linux capabilities.
- `NET_BIND_SERVICE` is not required.
- RBAC does not grant Secrets access, wildcard permissions, Deployment mutation, ConfigMap writes, or cluster-admin privileges.
- NetworkPolicy restricts egress to backend SSH and DNS.

Verification:

- `make helm-lint`: passed.
- `make helm-template`: passed.
- `make kustomize-build`: passed.
- `make test-security`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make build`: passed.

Notes:

- Helm emitted a local warning about an unrelated `helm-secrets` plugin configuration, but chart linting completed with `0 chart(s) failed`.

## Slice 8 Result

Implemented CI/CD and release automation artifacts:

- GitHub Actions workflows:
  - `ci.yml`
  - `security-scan.yml`
  - `performance.yml`
  - `release.yml`
- Renovate configuration:
  - Go modules.
  - GitHub Actions.
  - Dockerfiles.
- Makefile release and CI/CD targets:
  - `test-performance-local`
  - `container-build`
  - `helm-package`
  - `sbom`
  - `sign`
  - `release-check`
- Gateway HAProxy Dockerfile.
- `.gitignore` entries for generated artifacts.

Security and supply-chain assertions:

- workflows invoke Makefile targets rather than duplicating command logic.
- PR and scheduled workflows use `contents: read`.
- release workflow grants only required elevated permissions:
  - `contents: write`
  - `packages: write`
  - `id-token: write`
  - `attestations: write`
- release workflow includes SBOM, signing, and provenance generation steps.
- generated build and release artifacts are ignored.

Verification:

- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make helm-lint`: passed.
- `make helm-template`: passed.
- `make kustomize-build`: passed.
- `make build`: passed.

Notes:

- Helm emitted a local warning about an unrelated `helm-secrets` plugin configuration, but chart linting completed with `0 chart(s) failed`.
- Local `sbom`, `sign`, and `container-build` targets are tool-gated so developer machines without Syft, Cosign, or Docker can still run validation slices.

## End-to-End Validation Result

Implemented E2E validation foundation artifacts:

- `make test-e2e`.
- Process-level E2E test that builds the CLI and executes the real binary.
- E2E coverage for:
  - `dev-connect connect dev01 --no-code --no-reconnect --output json` with temporary config.
  - invalid config rejection before side effects.
  - `dev-connect config location`.
- CI workflow now runs `make test-e2e`.
- `make test` includes `test-e2e`.
- `make release-check` includes `test-e2e`.

Verification:

- `make test-e2e`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make test`: passed.
- `make helm-lint`: passed.
- `make helm-template`: passed.
- `make kustomize-build`: passed.
- `make build`: passed.

Notes:

- E2E validation uses temporary config and a locally built binary.
- E2E validation still avoids real `kubectl`, SSH, VS Code, and network calls.

## Connect Orchestration Foundation Result

Implemented connect preparation artifacts:

- Central `hostKeys` configuration inventory.
- Host key reference validation when a host key inventory is configured.
- `internal/connect` orchestration package.
- Target resolution from configuration.
- Gateway resolution from target default or explicit gateway override.
- Pinned SSH host key lookup from the central inventory.
- Injected local port allocation for deterministic tests.
- Temporary SSH config and `known_hosts` generation through the approved SSH config package.
- Session state creation with:
  - session ID,
  - target,
  - gateway,
  - namespace,
  - Kubernetes context,
  - local port,
  - reconnect setting,
  - start timestamp.
- Failure handling for unknown targets, unknown gateways, missing host keys, and port allocation errors.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- The orchestration package remains side-effect limited to local files and injected port allocation.
- This slice still does not start real `kubectl`, SSH, or VS Code processes.
- Host key material is used only to write the session-scoped `known_hosts` file; no SSH private keys are generated or stored.

## CLI Connect Preparation Integration Result

Implemented CLI integration artifacts:

- `dev-connect connect <target>` now loads the approved YAML configuration and invokes the connect orchestration package.
- The CLI resolves:
  - target,
  - gateway override,
  - Kubernetes context,
  - pinned SSH host key,
  - local session directory,
  - session-scoped SSH directory.
- The command writes local session state and temporary SSH config files before returning success.
- JSON output now reports:
  - `status: Prepared`,
  - target server,
  - generated session ID,
  - allocated local port.
- E2E tests now execute the real CLI binary and verify generated local session and SSH artifacts.
- Test-only environment hooks isolate E2E validation from the local workstation:
  - `DEV_CONNECT_SESSION_DIR`,
  - `DEV_CONNECT_SSH_DIR`,
  - `DEV_CONNECT_TEST_LOCAL_PORT`.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- This slice still does not start real `kubectl port-forward`, SSH, or VS Code.
- E2E tests remain deterministic and do not require a real Kubernetes cluster, SSH server, VS Code installation, or network listener.

## Kubernetes Preflight Foundation Result

Implemented Kubernetes preflight artifacts:

- `kubectl version` command builder for API reachability validation.
- `internal/preflight` package for ordered Kubernetes validation.
- Preflight validation sequence:
  - verify Kubernetes API reachability through `kubectl version`,
  - verify RBAC through `kubectl auth can-i create pods/portforward`,
  - verify functional port-forward behavior through a temporary `kubectl port-forward` command.
- RBAC denial detection when `kubectl auth can-i` exits successfully but returns `no`.
- Functional validation failure when port-forward output does not indicate readiness.
- Tests verify exact kubectl command arguments, including kubeconfig, context, namespace, Service, and port mapping.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- The preflight package uses only the injected `kubectl.Runner`; it does not import Kubernetes or Rancher client libraries.
- This slice does not yet wire preflight into the CLI because the real external process runner is still intentionally separate.

## kubectl Executable Runner Foundation Result

Implemented kubectl process execution artifacts:

- `kubectl.ExecutableRunner` for invoking the locally installed `kubectl` binary.
- `kubectl.ResolveExecutable` for deterministic executable discovery from configured search paths.
- Capturing of stdout, stderr, and exit code.
- Mapping of non-zero kubectl exits to `kubectl.ExitError`.
- Base environment propagation to the kubectl process.
- Per-command environment overrides without mutating global operating system settings.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- The runner shells out to the local `kubectl` executable and still does not import Kubernetes or Rancher client libraries.
- Tests use local fake executables and do not require a real Kubernetes cluster, kubeconfig, or network access.

## CLI kubectl Preflight Integration Result

Implemented CLI preflight artifacts:

- `dev-connect connect <target>` now runs Kubernetes preflight before returning `Prepared`.
- The CLI uses `kubectl.ExecutableRunner` to invoke the local `kubectl` binary.
- `kubectl` executable discovery supports:
  - `DEV_CONNECT_KUBECTL_PATH` for deterministic tests and controlled overrides,
  - normal `PATH` discovery for user execution.
- Preflight uses the selected dev-connect context and cluster to resolve:
  - Kubernetes context,
  - kubeconfig path,
  - process-scoped proxy overrides.
- Preflight validates:
  - Kubernetes API reachability,
  - RBAC for `pods/portforward`,
  - functional `kubectl port-forward` readiness.
- If preflight fails after local preparation, generated session state and SSH files are cleaned up.
- CLI unit tests and E2E tests use a fake `kubectl` executable and verify the full CLI flow without a real cluster.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- The client still communicates with Kubernetes only by executing `kubectl`.
- No direct HTTP(S), Kubernetes, or Rancher client implementation was introduced.

## kubectl Port-Forward Readiness Probe Result

Implemented long-running kubectl readiness artifacts:

- `kubectl.ReadyRunner` interface for commands that become ready before process completion.
- `kubectl.ExecutableRunner.RunUntilReady` for streaming stdout/stderr from long-running kubectl commands.
- Readiness detection for `kubectl port-forward` output.
- Automatic termination of the temporary functional port-forward probe after readiness is detected.
- Exit-error mapping when kubectl fails before readiness.
- Preflight now uses `RunUntilReady` for functional port-forward validation when the runner supports it.
- Existing fake runner behavior remains supported for deterministic unit tests.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- This fixes the production behavior where real `kubectl port-forward` is a long-running process.
- The implementation still executes only the local `kubectl` binary and does not introduce direct Kubernetes API clients.

## VS Code Remote SSH Launcher Foundation Result

Implemented VS Code launch artifacts:

- `vscode.Launcher` interface.
- `vscode.ExecutableLauncher` for invoking the local VS Code Desktop launcher.
- Remote SSH argument generation for `code --remote ssh-remote+<target>`.
- Launcher exit error mapping with stderr capture.
- CLI integration:
  - `--no-code` continues to skip VS Code launch,
  - without `--no-code`, the CLI resolves and invokes VS Code after successful preflight,
  - `DEV_CONNECT_CODE_PATH` provides a deterministic process-local launcher override for tests and controlled environments.
- Unit tests verify VS Code launcher arguments and failure handling.
- CLI tests verify fake VS Code launcher execution without requiring a real VS Code installation.

Verification:

- `make test`: passed.
- `make lint`: passed with `golangci-lint` 2.12.2.
- `make build`: passed.

Notes:

- VS Code launch still uses VS Code Desktop, not browser-based VS Code.
- GitHub Copilot and local VS Code extensions remain local because the desktop `code` launcher is used.
