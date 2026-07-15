# dev-connect Implementation Plan

Status: Approved

## 1. Purpose

This document defines the first implementation phase after approval of Phases 1 through 10.

Implementation shall follow the approved Specification Driven Development and Test Driven Development process. Tests and validation scaffolding shall be created before production implementation code.

## 2. Implementation Principles

- Implementation shall follow the approved architecture, specifications, requirements, designs, and documentation.
- Work shall proceed in small vertical slices.
- Each slice shall start with tests or validation checks.
- CI/CD command logic shall be exposed through Makefile targets.
- GitHub Actions workflows shall later reuse Makefile targets.
- No direct Kubernetes or Rancher API client shall be introduced.
- All Kubernetes interactions shall execute through the local `kubectl` binary.
- Security and architecture compliance tests are mandatory gates.
- Initial Go baseline is Go 1.26.x.
- Initial Go module path and import root use `github.com/anwendt/dev-connect`.
- The first local test pass targets macOS while preserving cross-platform design boundaries.

## 3. First Implementation Slices

### Slice 1: Project Tooling Foundation

Purpose:

- Establish the Go module, Makefile targets, lint configuration, and dependency policy.
- Use module path `github.com/anwendt/dev-connect`.
- Use Go 1.26.x as the initial toolchain baseline.

Test-first artifacts:

- Makefile target validation.
- Static dependency license validation stub.
- Forbidden import validation test.

Implementation artifacts:

- `go.mod`
- `Makefile`
- `.golangci.yml`
- initial package directories
- `cmd/dev-connect`
- `internal/config`
- `internal/kubectl`
- `internal/proxy`
- `internal/session`
- `internal/sshconfig`
- `internal/vscode`
- `internal/logging`
- `internal/output`

Acceptance:

- `make help` works.
- `make fmt-check` works.
- `make lint` is wired.
- `make test-unit` is wired.
- forbidden Kubernetes/Rancher client import checks are executable.
- foundation tests pass on macOS.

### Slice 2: CLI Skeleton

Purpose:

- Provide the Cobra command structure without external side effects.

Test-first artifacts:

- CLI command parsing unit tests.
- `--output json` schema tests.
- `--log-format` independence tests.

Implementation artifacts:

- `dev-connect connect`
- `dev-connect disconnect`
- `dev-connect status`
- `dev-connect list`
- `dev-connect version`
- `dev-connect config location`

Acceptance:

- commands parse deterministically,
- JSON output includes `apiVersion`,
- command errors are testable and structured.

### Slice 3: Configuration Loading

Purpose:

- Implement YAML configuration loading and validation.

Test-first artifacts:

- config precedence tests,
- schema version rejection tests,
- OS default path tests,
- proxy override validation tests.

Implementation artifacts:

- YAML config loader using `sigs.k8s.io/yaml`,
- default path resolver,
- config validation package.

Acceptance:

- config paths follow OS conventions,
- invalid config fails before external processes start,
- proxy overrides are represented without global mutation.

### Slice 4: External Process Abstractions

Purpose:

- Add testable wrappers for `kubectl`, VS Code launcher, and process execution.

Test-first artifacts:

- fake `kubectl` helper tests,
- mock VS Code launcher tests,
- process environment redaction tests.

Implementation artifacts:

- `kubectl` detection,
- command construction,
- process-scoped proxy environment,
- VS Code discovery.

Acceptance:

- no real kubeconfig, real kubectl, real VS Code, or internet access is required for tests.

### Slice 5: SSH Config and Host Key Pinning

Purpose:

- Generate temporary SSH config and `known_hosts`.

Test-first artifacts:

- host key pinning tests,
- host key mismatch tests,
- host key rotation integration test,
- cleanup tests.

Implementation artifacts:

- temporary file generation,
- restrictive file permissions,
- pinned host key writer.

Acceptance:

- `StrictHostKeyChecking=no` is never generated,
- temporary files contain no private keys,
- cleanup removes session-scoped files.

### Slice 6: Tunnel Lifecycle

Purpose:

- Manage `kubectl port-forward` lifecycle.

Test-first artifacts:

- free port allocation tests,
- port-forward readiness tests with fake `kubectl`,
- reconnect tests,
- stale state tests.

Implementation artifacts:

- local port allocator,
- tunnel supervisor,
- JSON session state,
- lock handling.

Acceptance:

- one active managed session per user profile by default,
- reconnect is automatic by default,
- unrelated processes are not killed.

### Slice 7: Kubernetes Gateway Manifests

Purpose:

- Implement Kubernetes manifests, Helm chart, and Kustomize resources after tests are defined.

Test-first artifacts:

- YAML syntax validation,
- Kubernetes schema validation,
- RBAC static validation,
- NetworkPolicy validation,
- Pod Security validation,
- Helm template assertions.

Implementation artifacts:

- Namespace,
- Deployment,
- Service,
- ConfigMap,
- NetworkPolicy,
- PodDisruptionBudget,
- optional HPA,
- RBAC,
- Helm chart,
- Kustomize base and overlays.

Acceptance:

- no `LoadBalancer`, `NodePort`, or Ingress,
- Service port `22` targets container port `2222`,
- `automountServiceAccountToken: false`,
- RBAC remains least privilege.

### Slice 8: CI/CD and Release Automation

Purpose:

- Implement workflow YAML after Makefile targets exist.

Test-first artifacts:

- Makefile target coverage validation,
- workflow permissions review,
- generated artifact cleanliness check.

Implementation artifacts:

- GitHub Actions workflows,
- Renovate configuration,
- SBOM generation,
- Cosign signing,
- provenance generation.

Acceptance:

- workflows call Makefile targets,
- temporary artifacts are not committed,
- supply chain controls align with SLSA Level 3 target where feasible.

## 4. Required Order

1. Tooling foundation.
2. Unit test scaffolding.
3. CLI skeleton.
4. Configuration and output schemas.
5. External process abstractions and fake tools.
6. SSH host key handling.
7. Tunnel lifecycle.
8. Gateway manifests and packaging.
9. CI/CD workflows.
10. End-to-end validation.

## 5. Review Gate

Before writing production implementation code for a slice:

- the slice must have test cases or validation checks,
- the expected behavior must trace to approved specification sections,
- security constraints must be represented where applicable,
- Makefile target integration must be defined where applicable.
