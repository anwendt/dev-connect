# dev-connect Phase 7 Review

Status: Approved

## Scope

Phase 7 currently covers testing strategy only. It intentionally does not include Go test files, integration test harnesses, Kubernetes manifests, Helm charts, Kustomize files, CI/CD workflows, binaries, or container images.

Primary artifact:

- `testing-strategy.md`

## Review Checklist

TDD and scope:

- [x] TDD workflow is explicit.
- [x] Unit test scope is sufficient.
- [x] Integration test scope is sufficient.
- [x] End-to-end test scope is sufficient.
- [x] Traceability expectations are sufficient.

Mocks and harnesses:

- [x] Fake `kubectl` strategy is sufficient.
- [x] Mock SSH server strategy is sufficient.
- [x] Mock VS Code strategy is sufficient.
- [x] External process behavior can be simulated.

Security and architecture compliance:

- [x] RBAC static validation requirements are sufficient.
- [x] Direct Kubernetes/Rancher client import checks are sufficient.
- [x] `kubectl`-only architecture compliance checks are sufficient.
- [x] Host key rotation integration test requirements are sufficient.
- [x] Metadata log redaction tests are sufficient.
- [x] CI quality gates are sufficient.

Gateway and Kubernetes:

- [x] Gateway test scope is sufficient.
- [x] NetworkPolicy test scope is sufficient.
- [x] HPA/default replica tests are covered.
- [x] ServiceAccount token automount tests are covered.

Reliability and performance:

- [x] Failure tests are sufficient.
- [x] Reconnect tests are sufficient.
- [x] Concurrency tests are sufficient.
- [x] Performance test assumptions are sufficient.
- [x] 50-100 session load-test smoke test is represented.

## Required Decisions Before Creating Tests

The following decisions are approved for the initial testing architecture:

1. Go test framework:
   - use Go standard library `testing` as the primary test framework.
   - use `testify` for assertions.
   - use `testify/mock` only where necessary.
   - use `testing.B` for benchmarks.
   - do not introduce a BDD framework such as Ginkgo for the initial implementation.
2. Fake `kubectl`:
   - implement as a dedicated Go helper binary.
   - support configurable responses for `auth can-i`, `port-forward`, `version`, `config`, context selection, and failure scenarios.
3. Mock SSH server:
   - implement a lightweight Go-based mock SSH server using Go SSH libraries.
   - support successful authentication, failed authentication, host key validation, host key rotation, connection interruption, timeout simulation, and banner exchange.
4. Static validation of Kubernetes resources:
   - until Phase 8 repository structure exists, generate Kubernetes manifests into a temporary test directory during test execution.
   - static validation consumes generated manifests.
   - validation includes YAML syntax, Kubernetes schema validation, RBAC validation, NetworkPolicy validation, and Pod Security validation.
5. Performance tests:
   - local tests use `go test -bench`.
   - manual CI tests are manually triggered for PR validation and performance regression checks.
   - dedicated performance environment tests cover load testing, scalability validation, resource sizing, and capacity planning.
6. Test traceability:
   - every specification section shall have a unique identifier such as `SPEC-001`.
   - every test shall reference one or more specification identifiers.
   - traceability matrix shall be automatically generated from source code annotations.
7. Test data management:
   - all test data shall be stored under `testdata/`.
   - test data includes SSH keys, example kubeconfigs, proxy configs, YAML manifests, and fixtures.
8. Test isolation:
   - tests shall not use real `~/.kube/config`, real `kubectl`, real VS Code, internet connections, enterprise credentials, or production SSH keys.
   - external components shall be simulated or mocked.

Remaining decisions before creating tests:

No Phase 7 testing architecture decisions remain open.

## Approval Gate

Phase 7 is approved. Phase 8 may start.
