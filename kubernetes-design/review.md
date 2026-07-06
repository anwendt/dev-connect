# dev-connect Phase 4 Review

Status: Approved

## Scope

Phase 4 covers Kubernetes design only. It intentionally does not include actual Kubernetes YAML manifests, Helm chart files, Kustomize files, Go code, tests, container builds, or CI/CD workflows.

Primary artifact:

- `kubernetes-design.md`

## Review Checklist

Gateway model:

- [x] One HAProxy TCP listener on port 22 is correctly represented.
- [x] One backend target per gateway Deployment is correctly represented.
- [x] Multi-target routing is explicitly deferred.
- [x] Gateway remains TCP-only.
- [x] Gateway does not perform SSH authentication.

Kubernetes resources:

- [x] Namespace design is acceptable.
- [x] Deployment design is acceptable.
- [x] Service design is acceptable.
- [x] ConfigMap design is acceptable.
- [x] NetworkPolicy design is acceptable.
- [x] PodDisruptionBudget design is acceptable.
- [x] HorizontalPodAutoscaler design is acceptable.
- [x] ServiceAccount design is acceptable.

Security:

- [x] No public endpoint is introduced.
- [x] No SSH credentials are stored in Kubernetes.
- [x] Developer RBAC is least privilege.
- [x] Kubernetes communication is delegated to local `kubectl`; no direct Rancher or Kubernetes API client is introduced.
- [x] Optional proxy overrides are scoped only to `kubectl` processes started by `dev-connect`.
- [x] Operator RBAC is separated from developer access.
- [x] ServiceAccount token handling is acceptable.
- [x] NetworkPolicy remains CNI-independent.

Operations:

- [x] Initial resource requests and limits are acceptable.
- [x] Production replica defaults are acceptable.
- [x] Health check strategy is acceptable.
- [x] PDB behavior is acceptable.
- [x] HPA assumptions are acceptable.
- [x] Upgrade and rollback strategy is acceptable.

Observability:

- [x] Minimum telemetry signals are sufficient.
- [x] Azure Monitor / Log Analytics first-deployment assumption is represented.
- [x] Metadata-only logging is preserved.
- [x] Audit correlation requirements are preserved.

Packaging:

- [x] Kustomize design is acceptable.
- [x] Helm chart design is acceptable.
- [x] Values model is acceptable.
- [x] GitOps compatibility is acceptable.

## Required Decisions Before Phase 5

The following decisions are recorded and shall guide later manifest, Helm, and Kustomize implementation:

1. HAProxy container listens internally on non-privileged TCP port `2222`; Service exposes TCP port `22` and forwards to target port `2222`.
2. Replica count is environment-specific and configurable:
   - development: 1.
   - test/integration: 2.
   - production: 2.
3. HPA is disabled by default in the first implementation; the design supports enabling it later without redesign.
4. Backend addressing supports both DNS names and IP addresses; DNS names are recommended and preferred.
5. DNS egress is enabled only when DNS-based backend addressing is configured.
6. `automountServiceAccountToken` is disabled by default and enabled only for workloads that explicitly require Kubernetes API access.

Remaining decisions before Phase 5:

No Phase 4 Kubernetes design decisions remain open.

## Approval Gate

Phase 4 is approved. Phase 5 may start.
