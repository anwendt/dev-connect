# dev-connect Phase 3 Review

Status: Approved

## Scope

Phase 3 covers non-functional requirements only. It intentionally does not include Go code, tests, Kubernetes manifests, Helm chart files, container builds, or CI/CD definitions.

Primary artifact:

- `non-functional-requirements.md`

## Review Checklist

Performance:

- [x] Capacity target of 100 simultaneous developers is acceptable.
- [x] Growth target of 250 simultaneous developers without architectural redesign is acceptable.
- [x] Throughput assumptions are acceptable for performance testing.
- [x] Latency and responsiveness targets are acceptable.
- [x] Kubernetes API server load considerations are sufficient.

Security:

- [x] Rancher authentication and RBAC requirements are acceptable.
- [x] SSH host key pinning requirements are acceptable.
- [x] Secret handling requirements are acceptable.
- [x] Network security requirements are acceptable.
- [x] Supply chain security expectations are acceptable for later phases.

Maintainability:

- [x] Code maintainability requirements are sufficient for Go implementation.
- [x] Configuration maintainability requirements are sufficient.
- [x] Operational maintainability requirements are sufficient.
- [x] GitOps ownership expectations are clear.

Observability:

- [x] Metadata-only logging requirements are sufficient.
- [x] Azure Monitor / Log Analytics first-deployment assumption is acceptable.
- [x] Metrics requirements are sufficient.
- [x] Audit correlation requirements are sufficient.
- [x] Alerting expectations are sufficient.

Portability and compatibility:

- [x] Windows, Linux, and macOS client portability requirements are sufficient.
- [x] Kubernetes v1.30+ portability requirements are sufficient.
- [x] CNI-independent NetworkPolicy requirement is acceptable.
- [x] Rancher-managed cluster compatibility is sufficient.
- [x] VS Code, OpenSSH, and `kubectl` compatibility requirements are sufficient.

Recovery:

- [x] Client recovery requirements are sufficient.
- [x] Gateway recovery requirements are sufficient.
- [x] Automatic reconnect behavior is sufficiently constrained.
- [x] Session timeout cleanup requirements are acceptable.

Upgrade and versioning:

- [x] Client upgrade strategy is sufficient.
- [x] Gateway upgrade strategy is sufficient.
- [x] GitOps rollback expectations are sufficient.
- [x] Semantic versioning requirements are acceptable.
- [x] Configuration schema versioning requirements are acceptable.

## Required Decisions Before Phase 4

The following decisions are recorded and shall guide Phase 4 Kubernetes Design:

1. NFR latency targets are design goals, not hard acceptance thresholds for the first implementation.
2. Phase 7 shall include a formal pre-production load-test smoke test with 50-100 simultaneous sessions and representative developer activities.
3. Phase 4 shall define minimum telemetry signals for Kubernetes design and instrumentation, including tunnel startup duration, active sessions, error rates, and Gateway Pod CPU/RAM usage.
4. Detailed Azure Monitor / Log Analytics metric names and dashboard layouts may be refined after Phase 4.
5. Initial gateway Pod sizing uses fixed requests and limits:
   - request: 100 mCPU and 128 MiB memory.
   - limit: 500 mCPU and 256 MiB memory.
6. Phase 7 performance tests shall measure actual utilization and adjust gateway Pod sizing where needed.

Remaining decisions before Phase 4:

No Phase 3 non-functional decisions remain open.

## Approval Gate

Phase 3 is approved. Phase 4 may start.
