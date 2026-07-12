# dev-connect Testing Strategy

Status: Approved

## 1. Purpose

This document defines the testing strategy for `dev-connect`.

Phase 7 defines how tests shall be designed before implementation. It does not create Go tests, integration test harnesses, Kubernetes manifests, CI/CD workflows, or executable test code.

## 2. Testing Principles

- Follow Test Driven Development.
- Tests shall be written before implementation code.
- Security and architecture compliance tests are mandatory CI quality gates.
- Tests shall map back to approved architecture, functional, NFR, Kubernetes, client, and security specifications.
- Tests shall avoid real enterprise credentials, real SSH private keys, and real Kubernetes tokens.
- Tests shall be deterministic and isolated where practical.

## 3. Test Categories

Required test categories:

- Unit tests.
- Integration tests.
- End-to-end tests.
- Mock Kubernetes API behavior through `kubectl` process mocks.
- Mock SSH server tests.
- Mock VS Code launcher tests.
- Gateway tests.
- Port-forward tests.
- Failure tests.
- Reconnect tests.
- Security tests.
- Concurrency tests.
- Performance tests.

## 4. TDD Workflow

Each implementation increment shall follow:

1. Add or update specification traceability.
2. Write failing tests first.
3. Implement the minimum code required.
4. Run targeted tests.
5. Run broader affected test suite.
6. Update documentation if behavior changes.

No feature shall be considered complete without tests mapped to the approved requirements.

The testing architecture shall follow this order:

1. Specification.
2. Unit tests.
3. Integration tests.
4. Implementation.
5. Performance tests.
6. End-to-end tests.

Implementation shall never precede specification and test definition.

## 5. Test Framework

Approved test framework choices:

| Purpose | Library |
| --- | --- |
| Unit tests | Go standard library `testing` |
| Assertions | `testify` |
| Mocking | `testify/mock` only where necessary |
| Integration tests | Go standard library `testing` |
| Benchmark tests | `testing.B` |

No BDD framework, such as Ginkgo, shall be introduced for the initial implementation.

## 6. Unit Test Scope

Unit tests shall cover:

- CLI command parsing.
- Global flag validation.
- `--output json` response generation.
- `--log-format` behavior.
- YAML config parsing with `sigs.k8s.io/yaml`.
- Config validation.
- Proxy override environment construction.
- `kubectl` command construction.
- VS Code launcher discovery order.
- OS-specific config paths.
- Local port allocation.
- Session state JSON serialization.
- Locking behavior.
- SSH config generation.
- Temporary `known_hosts` generation.
- Redaction helpers.
- Exit code mapping.

## 7. Integration Test Scope

Integration tests shall cover:

- Managed `kubectl` process invocation using fake `kubectl` executables.
- `kubectl auth can-i` preflight success and failure.
- Temporary `kubectl port-forward` validation success and failure.
- Port-forward readiness parsing.
- Tunnel process lifecycle.
- Session state creation and cleanup.
- Stale session detection.
- Proxy override propagation only to `kubectl`.
- VS Code launcher invocation with fake launcher.
- Host key rotation behavior.

## 8. End-to-End Test Scope

End-to-end tests shall cover representative workflows:

- `dev-connect connect dev01` with fake `kubectl`, fake SSH server, and fake VS Code launcher.
- `dev-connect disconnect`.
- `dev-connect status`.
- `dev-connect list`.
- Tunnel-only mode with `--no-code`.
- JSON output mode with `--output json`.
- Reconnect after managed port-forward interruption.
- Timeout cleanup.

E2E tests shall not require real enterprise credentials.

## 9. Mock Kubernetes and kubectl Strategy

Because `dev-connect` must not directly connect to Rancher or the Kubernetes API, Kubernetes behavior shall be tested by mocking `kubectl`.

A dedicated Go helper binary shall implement fake `kubectl`.

The fake `kubectl` helper shall support configurable responses for:

- `auth can-i`,
- `port-forward`,
- `version`,
- `config`,
- context selection,
- failure scenarios.

Fake `kubectl` shall simulate:

- `kubectl version --client`.
- `kubectl auth can-i`.
- namespace checks.
- service discovery.
- endpoint or Pod readiness discovery.
- temporary `kubectl port-forward` validation.
- long-running managed `kubectl port-forward`.
- stdout/stderr readiness output.
- permission failures.
- admission failures.
- Rancher-specific authorization failures.
- process crashes.

Architecture compliance tests shall verify:

- no direct Kubernetes client imports,
- no Rancher client imports,
- no direct HTTP(S) calls to Rancher or Kubernetes API code paths,
- every Kubernetes operation goes through the `kubectl` package/process abstraction.

## 10. Mock SSH Server Strategy

A lightweight Go-based mock SSH server shall be implemented using Go SSH libraries.

The mock SSH server shall support:

- successful authentication,
- failed authentication,
- host key validation,
- host key rotation,
- connection interruption,
- timeout simulation,
- banner exchange.

Mock SSH tests shall verify:

- pinned host key accepted,
- unknown host key rejected,
- replaced host key rejected until inventory is updated,
- successful connection after approved host key rotation,
- obsolete host key cleanup,
- SSH authentication failures are not retried by `dev-connect`.

The gateway and client shall never inspect SSH credentials.

## 11. Mock VS Code Strategy

Fake VS Code launcher shall verify:

- discovery from `PATH`,
- fallback to OS-specific default paths,
- user-configured launcher path fallback,
- correct Remote SSH command arguments,
- no browser-based VS Code behavior,
- `--no-code` suppresses launcher execution.

## 12. Gateway Test Scope

Gateway tests shall verify later manifest/chart output for:

- HAProxy listens internally on port `2222`.
- Service exposes port `22` and forwards to target port `2222`.
- one backend target per gateway Deployment.
- no SSH termination.
- no public endpoint.
- non-root Pod security context.
- no `NET_BIND_SERVICE` requirement.
- `automountServiceAccountToken: false`.
- resource requests `100m` CPU and `128Mi` memory.
- resource limits `500m` CPU and `256Mi` memory.
- HPA disabled by default.
- environment-specific replica defaults.

Until the GitOps repository structure exists in Phase 8, Kubernetes manifests shall be generated into a temporary test directory during test execution. Static validation shall consume these generated manifests.

Static validation shall include:

- YAML syntax.
- Kubernetes schema validation.
- RBAC validation.
- NetworkPolicy validation.
- Pod Security validation.

## 13. RBAC Security Tests

Automated static validation shall inspect generated RBAC resources.

Forbidden unless explicitly approved:

- Secret reads: `get`, `list`, `watch`.
- ConfigMap write operations.
- Pod delete.
- Deployment modification.
- Cluster-wide wildcard permissions.
- Cluster-admin privileges.
- Privileged Pod creation.
- RBAC modification permissions.

CI shall fail on RBAC privilege creep.

## 14. NetworkPolicy Tests

NetworkPolicy tests shall verify:

- default deny is present.
- gateway egress is limited to approved backend TCP 22 destinations.
- DNS egress is enabled only when DNS backend addressing is configured.
- DNS egress is absent for static-IP-only backends unless explicitly required.
- metrics scraping is restricted to approved monitoring sources.
- no CNI-specific policy extensions are required.

## 15. Log Redaction Tests

Every CLI command producing metadata logs shall have redaction tests.

Tests shall verify logs never contain:

- passwords,
- SSH private keys,
- bearer tokens,
- Kubernetes tokens,
- proxy credentials,
- environment secrets,
- kubeconfig credentials,
- authentication headers,
- SSH payloads,
- terminal contents,
- transferred file contents.

Allowed metadata includes:

- username where available,
- target server alias,
- timestamps,
- session ID,
- Kubernetes context,
- namespace,
- gateway service,
- local port,
- exit code.

## 16. Failure Tests

Failure tests shall cover:

- missing `kubectl`.
- invalid config.
- invalid kube context.
- missing RBAC.
- failed `kubectl auth can-i`.
- failed temporary port-forward validation.
- gateway Service missing.
- gateway endpoint unavailable.
- local port collision.
- port-forward readiness timeout.
- VS Code launcher missing.
- SSH host key mismatch.
- SSH authentication failure.
- cleanup failure.

## 17. Reconnect Tests

Reconnect tests shall cover:

- managed port-forward process exits unexpectedly.
- bounded exponential backoff.
- max attempts exhausted.
- local port reuse when safe.
- no SSH credential retry.
- no reconnect after authorization failure.
- no reconnect after maximum session duration is reached.

## 18. Concurrency Tests

Concurrency tests shall cover:

- two simultaneous `connect` commands.
- `disconnect` during connect startup.
- stale lock cleanup.
- status while reconnect is active.
- cleanup idempotency.
- avoiding termination of unrelated `kubectl` processes.

## 19. Performance Tests

Performance tests shall be divided into three categories.

Local performance tests:

- Executed by developers with `go test -bench`.
- Purpose is fast feedback during development.

Manual CI performance tests:

- Triggered manually.
- Purpose is pull request validation and performance regression detection.

Dedicated performance environment tests:

- Executed on a dedicated Kubernetes environment.
- Purpose is load testing, scalability validation, resource sizing, and capacity planning.

Performance tests shall include:

- formal pre-production load-test smoke test with 50-100 simultaneous sessions.
- representative developer activity profiles:
  - terminal,
  - VS Code editing,
  - Git pull/push,
  - debugging,
  - file synchronization.
- average throughput baseline: 0.5 Mbit/s per developer.
- peak throughput baseline: 25 Mbit/s per developer.
- gateway CPU and memory measurements.
- Kubernetes API server port-forward load observations.
- reconnect behavior under transient gateway disruption.

The 100-developer target shall be validated before production rollout where platform access allows.

Performance tests shall not execute automatically on every CI pipeline because they are time-consuming and infrastructure-dependent.

## 20. Test Data Management

All test data shall be stored under `testdata/` directories.

Test data includes:

- SSH host keys for tests.
- Example kubeconfigs.
- Proxy configurations.
- YAML manifests.
- JSON session state examples.
- CLI output fixtures.

Test data shall not contain real enterprise credentials, real SSH private keys, real Kubernetes tokens, or production kubeconfig content.

## 21. Test Isolation

Every test shall be independent from the local development environment.

Tests shall not use:

- the real `~/.kube/config`,
- the real installed `kubectl`,
- a real installed VS Code,
- network connections to the internet,
- enterprise credentials,
- production SSH keys.

External components shall be simulated or mocked:

- `kubectl`,
- SSH server,
- VS Code launcher,
- filesystem where practical,
- clocks/timers where needed.

Tests shall be reproducible on developer workstations and in CI.

## 22. CI Quality Gates

Pull requests shall fail if:

- RBAC permissions exceed the approved security model.
- forbidden Kubernetes or Rancher client libraries are imported.
- Kubernetes operations bypass `kubectl`.
- sensitive information is written to logs.
- host key validation deviates from the approved security model.
- required unit tests fail.
- required integration tests fail.
- generated gateway manifests violate security expectations.

## 23. Traceability

Every specification section shall have a unique identifier.

Example:

```text
SPEC-001
SPEC-002
SPEC-003
```

Every test shall reference one or more specification identifiers.

Example:

```text
TEST-UNIT-001 -> SPEC-001
TEST-INT-014 -> SPEC-006
TEST-E2E-003 -> SPEC-015
```

The traceability matrix shall be automatically generated from source code annotations.

Example:

```go
// SPEC-004
func TestPortForwardConnection(t *testing.T) {}
```

Each test should map to:

- requirement source document,
- requirement section,
- test category,
- expected behavior,
- failure mode.

Traceability shall support audit, maintenance, coverage reporting, and Specification Driven Development.

## 24. Phase Gate

This document completes the approved Phase 7 testing strategy. Phase 8 may start.
