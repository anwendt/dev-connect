# dev-connect Security Specification

Status: Approved

## 1. Purpose

This document defines the security requirements and controls for `dev-connect`.

Phase 6 consolidates and sharpens the security model from the approved architecture, functional specification, non-functional requirements, Kubernetes design, and client design. It does not create Go code, tests, Kubernetes manifests, Helm charts, Kustomize files, container images, or CI/CD workflows.

## 2. Security Principles

- Least privilege is mandatory.
- The gateway shall only forward TCP traffic.
- The gateway shall not perform SSH authentication.
- SSH authentication shall always remain on the remote Linux development server.
- No public endpoints are allowed.
- No VPN or bastion host is introduced.
- No SSH private keys, SSH credentials, user passwords, Kubernetes tokens, or user databases shall be stored in Kubernetes.
- `dev-connect` shall not connect directly to Rancher or the Kubernetes API.
- All Kubernetes communication shall be delegated to the locally installed `kubectl` binary.
- Logs and audit records shall contain metadata only.
- SSH host key pinning is mandatory.

## 3. Identity and Authentication

Kubernetes:

- Authentication and authorization shall follow the existing Rancher authentication model.
- The solution shall not introduce a separate identity store.
- Access shall be granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.
- Local Kubernetes users are not part of the `dev-connect` access model.
- `kubectl` shall use the user's existing kubeconfig and enterprise identity flow.

SSH:

- SSH authentication shall be performed only by the target Linux development server.
- The gateway shall not inspect or validate SSH credentials.
- The gateway shall not terminate SSH.
- The gateway shall not store SSH keys or user records.

Client:

- The client shall not store passwords, Kubernetes tokens, SSH private keys, or kubeconfig contents.
- Optional proxy overrides shall apply only to `kubectl` child processes started by `dev-connect`.
- Proxy overrides shall not modify global OS, corporate, shell, or kubeconfig proxy configuration.

## 4. Authorization and RBAC

Developer RBAC shall be namespace-scoped and least privilege.

Allowed developer permissions:

- Read gateway discovery resources in the `dev-connect` namespace:
  - `services`: `get`, `list`.
  - selected `pods`: `get`, `list`.
  - endpoint resources required by supported `kubectl port-forward` behavior: `get`, `list`.
- Create port-forward sessions:
  - `pods/portforward`: `create`.
- Optional preflight:
  - `selfsubjectaccessreviews.authorization.k8s.io`: `create`, where required and permitted by cluster policy.

Forbidden developer permissions:

- Read Secrets.
- Create Pods.
- Update ConfigMaps.
- Exec into Pods.
- Update Deployments.
- Access unrelated namespaces.

Operator RBAC:

- Operators may manage gateway resources through GitOps and approved platform workflows.
- Operator permissions shall be separate from developer permissions.
- Operators shall not manage SSH user credentials in Kubernetes.

## 5. Network Security

Gateway exposure:

- Service type shall be `ClusterIP`.
- No `LoadBalancer`.
- No `NodePort`.
- No Ingress.
- No public IP.

NetworkPolicy:

- The solution shall rely only on the Kubernetes `NetworkPolicy` API.
- Default deny ingress and egress shall be used for the `dev-connect` namespace.
- Gateway egress shall be restricted to approved backend development server destinations on TCP port 22.
- DNS egress shall be enabled only when DNS-based backend addressing is configured.
- Metrics scraping shall be allowed only from approved monitoring namespaces or scraper Pods.

Private cloud:

- Private cloud firewall rules should allow TCP 22 only from approved gateway source ranges.
- Development servers shall not be directly exposed to developer workstations.

## 6. Gateway Security

HAProxy:

- HAProxy shall run as a TCP forwarder only.
- HAProxy shall listen internally on non-privileged TCP port `2222`.
- Kubernetes Service shall expose port `22` and forward to container port `2222`.
- The container shall not require root or `NET_BIND_SERVICE`.
- HAProxy shall not terminate SSH.
- HAProxy shall not inspect SSH credentials.
- HAProxy shall not log payloads.

Pod security:

- Run as non-root.
- Disable privilege escalation.
- Drop Linux capabilities unless explicitly justified.
- Use seccomp `RuntimeDefault`.
- Use read-only root filesystem where practical.
- `automountServiceAccountToken` shall be disabled by default.
- Gateway Pods shall not require Kubernetes API access at runtime.

## 7. Client Security

Temporary files:

- Temporary SSH config shall be generated per session.
- Temporary `known_hosts` shall be generated per session from pinned host keys.
- Temporary files shall be scoped to the current OS user.
- Temporary files shall use restrictive permissions where supported.
- Temporary files shall be deleted during cleanup.

Session state:

- Session state shall be JSON.
- Session state shall not contain credentials.
- Stale state shall be cleaned safely.
- One active managed session per user profile is the first-release default.

External process handling:

- `kubectl` process arguments and environment shall be constructed safely.
- Proxy overrides shall be process-scoped.
- The client shall not kill unrelated `kubectl` processes.
- VS Code launcher discovery shall avoid executing untrusted paths unless explicitly configured by the user.

## 8. SSH Host Key Security

Requirements:

- SSH host key pinning is mandatory.
- `StrictHostKeyChecking=no` is forbidden.
- `accept-new` shall not be the default.
- The client shall load expected host keys from centrally managed infrastructure configuration.
- Pinned SSH host keys are owned by the existing Platform GitOps repository under a dedicated `dev-connect` directory.
- Host key changes require the standard platform pull request approval workflow.
- The client shall write expected host keys to a temporary `known_hosts` file before launching VS Code Remote SSH.

Rotation:

1. Generate new host key on the target development server.
2. Update the Platform GitOps `dev-connect` host key inventory.
3. Review and approve through the standard platform pull request workflow.
4. Deploy updated inventory.
5. Client uses the new pinned key on next connection.

## 9. Logging and Auditing Security

Logging:

- Logs shall contain metadata only.
- Logs shall not include SSH payloads, terminal contents, transferred file contents, passwords, tokens, private keys, or full kubeconfig content.
- Logs shall include session ID, target, user identity where available, gateway, namespace, Kubernetes context, local port, start/end time, duration, and exit code.

Audit:

- Kubernetes audit logs should capture port-forward requests where platform policy supports it.
- Audit correlation shall be possible across client session ID, Kubernetes audit records, gateway metadata logs, and target server SSH logs.
- Azure Monitor / Log Analytics is the first deployment logging backend.
- Retention and SIEM forwarding are handled by the logging backend, not by `dev-connect`.

## 10. Supply Chain Security

Dependencies:

- External dependencies shall be minimized.
- Go standard library and Kubernetes SIG ecosystem libraries are preferred where suitable.
- Mandatory runtime dependencies shall use Apache License 2.0, MIT License, BSD-2-Clause, or BSD-3-Clause.

Approved first implementation choices:

- CLI framework: Cobra.
- YAML: `sigs.k8s.io/yaml`.
- Logging: Go standard library `log/slog`.

Later CI/CD controls:

- Dependency scanning.
- SBOM generation.
- Container scanning.
- Container signing.
- Helm linting.
- Static analysis.

## 11. Security Testing Requirements for Phase 7

Security and architectural compliance tests shall be mandatory CI quality gates.

### 11.1 RBAC Static Validation

Security tests shall include automated static validation of all generated Kubernetes RBAC resources.

The test suite shall verify that no generated Role or ClusterRole grants permissions beyond those explicitly required.

Forbidden permissions unless explicitly approved:

- Secrets: `get`, `list`, `watch`.
- ConfigMaps: write operations.
- Pods: `delete`.
- Deployments: modify operations.
- Cluster-wide wildcard permissions: `*`.
- Cluster-admin privileges.
- Privileged Pod creation.
- RBAC modification permissions.

### 11.2 Kubernetes Client Usage Verification

The Go client shall not directly communicate with Rancher or the Kubernetes API.

Automated static analysis shall verify that no direct Kubernetes or Rancher client libraries are imported by the client implementation.

Forbidden imports include, but are not limited to:

- `k8s.io/client-go`.
- `controller-runtime`.
- dynamic Kubernetes client libraries.
- Rancher API clients.

The client shall exclusively interact with Kubernetes through the locally installed `kubectl` executable.

Automated architecture compliance tests shall verify that every Kubernetes operation executed by the client is performed through the `kubectl` binary. No direct HTTP(S) communication to Rancher or the Kubernetes API server shall be performed by the client implementation.

### 11.3 Host Key Rotation Integration Test

Host key rotation shall be covered by an automated integration test.

The integration test shall verify:

- replacement of an existing host key,
- rejection of unknown host keys,
- successful connection after approved key rotation,
- cleanup of obsolete host keys.

Operational documentation shall additionally describe the enterprise host key rotation process.

### 11.4 Metadata Log Redaction Tests

Every CLI command producing metadata logs shall be covered by automated redaction tests.

The tests shall verify that logs never contain sensitive information, including:

- passwords,
- SSH private keys,
- bearer tokens,
- Kubernetes tokens,
- proxy credentials,
- environment secrets,
- kubeconfig credentials,
- authentication headers.

Structured metadata such as usernames, target servers, timestamps, and session identifiers may be logged where explicitly defined by the architecture.

### 11.5 Mandatory CI Failure Conditions

A pull request shall fail if:

- RBAC permissions exceed the approved security model,
- forbidden Kubernetes or Rancher client libraries are introduced,
- sensitive information is written to logs,
- host key validation behavior deviates from the approved security model,
- Kubernetes operations bypass the `kubectl` binary.

## 12. Threat Controls

| Threat | Control |
| --- | --- |
| Unauthorized tunnel access | Rancher-managed RBAC, namespace scoping, `pods/portforward` only. |
| Credential exposure in Kubernetes | No SSH keys, passwords, tokens, or user database in Kubernetes. |
| Gateway lateral movement | Egress NetworkPolicy, private firewall restrictions, non-root container. |
| MITM against SSH | Mandatory host key pinning and end-to-end SSH encryption. |
| API server overload | One port-forward per developer by default, monitoring, load tests. |
| Proxy misconfiguration | Process-scoped proxy overrides only. |
| Log data leakage | Metadata-only logging and redaction. |
| Compromised ServiceAccount token | Token automount disabled by default. |
| Unsafe container privileges | Non-root, no `NET_BIND_SERVICE`, drop capabilities, no privilege escalation. |

## 13. Security Acceptance Criteria

Phase 6 is acceptable when:

- Least privilege is explicitly defined for developers and operators.
- RBAC denies Secret read, exec, Pod create, ConfigMap update, and unrelated namespace access for developers.
- NetworkPolicies are defined as required controls.
- No public endpoint is introduced.
- No credentials are stored in Kubernetes.
- Gateway performs TCP forwarding only.
- SSH authentication remains on the remote Linux server.
- SSH host key pinning is mandatory.
- `dev-connect` delegates Kubernetes communication to `kubectl`.
- Logging and auditing remain metadata-only.
- Supply chain expectations are defined.
- Phase 7 security and architectural compliance test requirements are defined.

## 14. Phase Gate

This document completes the approved Phase 6 security specification. Phase 7 may start.
