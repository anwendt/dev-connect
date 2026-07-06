# dev-connect Phase 2 Review

Status: Approved

## Scope

Phase 2 covers functional behavior only. It intentionally does not include Go code, tests, Kubernetes manifests, Helm chart files, container builds, or CI/CD definitions.

Primary artifact:

- `functional-specification.md`

## Review Checklist

Developer workflow:

- [x] `dev-connect connect <target>` workflow is complete and correct.
- [x] `disconnect`, `status`, and `list` behavior is sufficient.
- [x] VS Code Desktop Remote SSH launch behavior is acceptable.
- [x] GitHub Copilot remains local and is not moved into Kubernetes or the remote server.
- [x] VS Code Server installation remains owned by VS Code Remote SSH.

Kubernetes behavior:

- [x] `kubectl` detection behavior is acceptable.
- [x] Kubernetes connectivity preflight is acceptable.
- [x] RBAC preflight behavior is acceptable.
- [x] Gateway discovery behavior is acceptable.
- [x] Port-forward process lifecycle is acceptable.

Gateway routing:

- [x] One-listener gateway model is preserved.
- [x] Explicit TCP routing requirement is clear enough for Phase 4 design.
- [x] Gateway remains TCP-only and does not perform SSH authentication.
- [x] Future CRD/operator inventory direction is preserved.

Client behavior:

- [x] YAML configuration model is acceptable.
- [x] Multiple contexts, clusters, and gateways are represented.
- [x] Temporary SSH config behavior is acceptable.
- [x] Local port allocation behavior is acceptable.
- [x] Session state model is acceptable.
- [x] Automatic reconnect defaults are acceptable.
- [x] Cleanup behavior is safe and idempotent.

Security:

- [x] No SSH private keys, Kubernetes tokens, passwords, or user databases are stored by the gateway.
- [x] Temporary files are restricted and removed.
- [x] Local listener binds only to loopback.
- [x] Required Kubernetes permissions are least privilege.
- [x] Logs and errors avoid leaking secrets.

Testability:

- [x] Functional requirements are testable.
- [x] Acceptance criteria are specific enough for TDD.
- [x] Error categories and exit codes are sufficient.
- [x] Mock Kubernetes API, mock SSH server, and mock VS Code behavior can be derived from the spec.

## Required Decisions Before Implementation

The following decisions are already recorded from the architecture update:

- SSH host key pinning is mandatory.
- `StrictHostKeyChecking=no` is forbidden.
- First release default is one active managed session and one active port-forward per developer.
- Kubernetes authentication and authorization follow the existing Rancher authentication model.
- Access is granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.
- The solution introduces no separate identity store.
- Logs and audits contain metadata only, not SSH contents.
- Log retention is enforced by the centralized enterprise logging backend, not by `dev-connect`.
- Development placeholder values are 30 days for metadata logs and 90 days for Kubernetes audit logs; they are not architectural constraints.
- Host keys are centrally managed as infrastructure configuration through the existing GitOps process.
- Host key changes require pull request approval by at least two authorized platform administrators before deployment.
- The client writes the expected host key to a temporary `known_hosts` file during connection startup.
- Session timeout defaults are 60 minutes idle timeout and 12 hours maximum session duration.
- Performance testing uses 0.5 Mbit/s average and 25 Mbit/s peak throughput per developer.
- Phase 4 implements one HAProxy TCP listener on port 22 with one backend target per gateway Deployment.
- Multi-target routing is deferred to a later phase using separate gateway Deployments per target or dynamic gateway generation.
- The CLI supports `--output json` for machine-readable command responses, independent from internal log format.
- The VS Code launcher is resolved from `PATH` first, then documented OS-specific default installation paths.
- Pinned SSH host keys are owned by the existing Platform GitOps repository under a dedicated `dev-connect` directory.
- Host key changes use the standard platform pull request approval workflow.
- The target platform logging backend for the first deployment is Azure Monitor / Log Analytics.
- Retention and forwarding to SIEM are handled by the logging backend, not by `dev-connect`.
- `dev-connect` never establishes direct network connections to Rancher or the Kubernetes API.
- All Kubernetes communication is delegated to the locally installed `kubectl` binary.
- `kubectl` uses the user's existing kubeconfig and enterprise proxy configuration by default.
- Optional user-specific proxy overrides apply only to `kubectl` processes started by `dev-connect`.
- Proxy overrides do not modify global operating system, corporate, shell, or kubeconfig proxy configuration.

Remaining decisions before writing tests or implementation:

No Phase 2 functional decisions remain open.

## Approval Gate

Phase 2 is approved. Phase 3 may start.
