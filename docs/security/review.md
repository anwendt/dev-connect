# dev-connect Phase 6 Review

Status: Approved

## Scope

Phase 6 covers security requirements and controls only. It intentionally does not include Go implementation files, tests, binaries, CI/CD workflows, Kubernetes manifests, Helm charts, Kustomize files, or container images.

Primary artifact:

- `security.md`

## Review Checklist

Identity and authorization:

- [x] Rancher authentication model is preserved.
- [x] No separate identity store is introduced.
- [x] Developer RBAC is least privilege.
- [x] Operator RBAC is separated from developer access.
- [x] Forbidden developer permissions are explicit.

Credential handling:

- [x] No SSH private keys are stored in Kubernetes.
- [x] No passwords are stored in Kubernetes.
- [x] No Kubernetes tokens are written by `dev-connect`.
- [x] No user database is stored in Kubernetes.
- [x] Client session state excludes credentials.

Network security:

- [x] No public endpoint is introduced.
- [x] ClusterIP-only gateway exposure is preserved.
- [x] NetworkPolicy controls are explicit.
- [x] DNS egress is conditional.
- [x] Private cloud firewall expectations are explicit.

Gateway security:

- [x] Gateway remains TCP-only.
- [x] Gateway does not perform SSH authentication.
- [x] HAProxy non-root and port `2222` design is preserved.
- [x] ServiceAccount token automount is disabled by default.
- [x] Pod security expectations are explicit.

Client security:

- [x] Kubernetes communication is delegated to `kubectl`.
- [x] Direct Rancher/Kubernetes API client use is forbidden.
- [x] Proxy overrides are process-scoped.
- [x] Temporary SSH config and `known_hosts` controls are explicit.
- [x] Cleanup requirements are explicit.

SSH host key security:

- [x] Host key pinning is mandatory.
- [x] `StrictHostKeyChecking=no` is forbidden.
- [x] Host key GitOps ownership is explicit.
- [x] Rotation workflow is explicit.

Logging, audit, and supply chain:

- [x] Metadata-only logging is explicit.
- [x] Audit correlation requirements are explicit.
- [x] Azure Monitor / Log Analytics first-deployment assumption is represented.
- [x] Dependency license policy is represented.
- [x] Future CI/CD security controls are listed.

## Required Decisions Before Phase 7

The following decisions are approved for Phase 7 testing design:

1. RBAC security validation:
   - automated static validation of all generated Kubernetes RBAC resources is required.
   - no generated Role or ClusterRole may grant permissions beyond those explicitly required.
   - forbidden permissions include Secret reads, ConfigMap writes, Pod deletes, Deployment modifications, wildcard permissions, cluster-admin privileges, privileged Pod creation, and RBAC modification permissions unless explicitly approved.
2. Kubernetes client usage verification:
   - automated static analysis shall verify that no direct Kubernetes or Rancher client libraries are imported by the Go client.
   - forbidden imports include `k8s.io/client-go`, `controller-runtime`, dynamic Kubernetes client libraries, and Rancher API clients.
   - all Kubernetes operations shall be executed through the local `kubectl` binary.
   - no direct HTTP(S) communication to Rancher or the Kubernetes API server is allowed from the client implementation.
3. Host key rotation testing:
   - host key rotation shall be covered by an automated integration test.
   - the test shall verify replacement of an existing host key, rejection of unknown host keys, successful connection after approved rotation, and cleanup of obsolete host keys.
   - operational documentation shall also describe the enterprise rotation process.
4. Metadata log redaction:
   - every CLI command producing metadata logs shall have automated redaction tests.
   - tests shall verify logs never contain passwords, SSH private keys, bearer tokens, Kubernetes tokens, proxy credentials, environment secrets, kubeconfig credentials, or authentication headers.
5. Mandatory CI quality gates:
   - pull requests shall fail on RBAC privilege creep, forbidden client imports, sensitive log output, host key validation regressions, or Kubernetes operations that bypass `kubectl`.

Remaining decisions before Phase 7:

No Phase 6 security decisions remain open.

## Approval Gate

Phase 6 is approved. Phase 7 may start.
