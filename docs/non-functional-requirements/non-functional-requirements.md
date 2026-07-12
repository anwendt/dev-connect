# dev-connect Non-Functional Requirements

Status: Approved

## 1. Purpose

This document defines the non-functional requirements for `dev-connect`.

Phase 3 builds on the approved Phase 1 architecture and approved Phase 2 functional specification. It does not introduce implementation code, tests, Kubernetes manifests, Helm charts, or CI/CD workflows.

## 2. Scope

In scope:

- Performance.
- Security.
- Maintainability.
- Observability.
- Portability.
- Compatibility.
- Recovery.
- Upgrade strategy.
- Versioning.

Out of scope:

- Kubernetes object design. Covered in Phase 4.
- Go client implementation. Covered in Phase 5.
- Test implementation. Covered in Phase 7.
- CI/CD implementation. Covered in Phase 9.

## 3. NFR Principles

- The gateway shall remain TCP-only and shall not perform SSH authentication.
- SSH authentication shall remain on the remote Linux development server.
- All SSH traffic shall traverse the Kubernetes API through `kubectl port-forward`.
- `dev-connect` shall never establish direct network connections to Rancher or the Kubernetes API.
- All Kubernetes communication shall be delegated to the locally installed `kubectl` binary.
- `kubectl` shall use the user's existing kubeconfig and enterprise proxy configuration by default.
- The solution shall not introduce VPN, bastion host, public endpoint, browser-based VS Code, or development workloads inside Kubernetes.
- `dev-connect` shall not store SSH private keys, Kubernetes tokens, passwords, or user databases.
- Kubernetes authentication and authorization shall follow the existing Rancher authentication model.
- The solution shall use Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.
- Pinned SSH host keys shall be managed through the existing Platform GitOps repository under a dedicated `dev-connect` directory.
- Logging and auditing shall record metadata only and never SSH session contents.

## 4. Performance

### 4.1 Capacity Targets

The first production design target shall support:

- 100 simultaneous developers.
- One active `kubectl port-forward` per developer by default.
- One HAProxy TCP listener on port 22 per gateway Deployment.
- One backend development server target per gateway Deployment.

The growth target shall support:

- 250 simultaneous developers without architectural redesign.
- Horizontal gateway scaling.
- Future multi-target support through separate gateway Deployments per target or dynamic gateway generation.

### 4.2 Throughput Assumptions

Performance testing shall model typical Remote SSH developer behavior.

| Activity | Average | Peak |
| --- | --- | --- |
| Terminal | < 50 kbit/s | 200 kbit/s |
| VS Code editing | 100-300 kbit/s | 1 Mbit/s |
| Git pull/push | Few Mbit/s | 20-50 Mbit/s |
| Debugging | < 1 Mbit/s | 5 Mbit/s |
| File synchronization | Few Mbit/s | 20-100 Mbit/s |

Baseline load tests shall use:

- 0.5 Mbit/s average throughput per developer.
- 25 Mbit/s peak throughput per developer.

### 4.3 Latency and Responsiveness

The target user experience shall remain suitable for interactive VS Code Remote SSH work.

Requirements:

- Tunnel startup should complete within 10 seconds after successful Kubernetes preflight under normal platform conditions.
- Gateway readiness checks should complete within 5 seconds under normal platform conditions.
- CLI commands that do not start a tunnel, such as `status`, `list`, and `version`, should return within 2 seconds when local state is healthy.
- Reconnect should start within 5 seconds of detecting an unexpected managed `kubectl port-forward` exit.

These are target thresholds for design and test planning, not hard guarantees across all enterprise network conditions.

Latency targets are design goals for the first implementation. They provide an architectural budget to identify bottlenecks early. Phase 7 performance tests may convert these targets into concrete acceptance thresholds after measured data is available.

### 4.4 Kubernetes API Load

The Kubernetes API server carries SSH traffic because port-forward streams traverse the API.

Requirements:

- The solution shall assume at most one active port-forward per developer by default.
- API server port-forward load shall be monitored.
- Load tests shall include 100 simultaneous sessions.
- Growth validation shall include a path to 250 simultaneous sessions.
- The platform shall define operational alert thresholds for API server saturation before production rollout.

Before the first production rollout, Phase 7 shall include a formal load-test smoke test that approximates expected traffic with 50-100 simultaneous sessions and representative developer activities. The test shall cover terminal usage, VS Code editing, Git pull/push, debugging, and file synchronization behavior.

## 5. Security

### 5.1 Identity and Access

Requirements:

- Kubernetes authentication and authorization shall follow the existing Rancher authentication model.
- The solution shall not introduce a separate identity store.
- Access shall be granted through Rancher-managed Kubernetes RBAC.
- Existing enterprise identity providers shall be used where configured in Rancher.
- Local Kubernetes users are not part of the `dev-connect` access model.
- Developer permissions shall be least privilege.
- Developers shall not need permission to read Secrets, exec into Pods, create Pods, update ConfigMaps, or access unrelated namespaces.
- `dev-connect` shall not embed or use a direct Kubernetes API client or Rancher API client.
- Optional proxy overrides shall be scoped only to `kubectl` child processes started by `dev-connect`.
- Proxy overrides shall not modify global operating system, corporate, shell, or kubeconfig proxy configuration.

### 5.2 SSH Security

Requirements:

- The gateway shall not terminate SSH.
- The gateway shall not perform SSH authentication.
- The gateway shall not inspect SSH credentials.
- SSH authentication shall always occur on the target Linux development server.
- `StrictHostKeyChecking=no` shall be forbidden.
- SSH host key pinning shall be mandatory.
- The client shall write expected pinned host keys to a temporary `known_hosts` file during connection startup.
- Host key changes shall follow the standard platform pull request approval workflow.

### 5.3 Secret Handling

Requirements:

- No SSH private keys in Kubernetes.
- No user passwords in Kubernetes.
- No Kubernetes tokens written by `dev-connect`.
- No user database in Kubernetes.
- Temporary files shall be scoped to the current OS user.
- Temporary files shall use restrictive permissions where supported by the operating system.
- Logs shall redact tokens, passwords, private keys, kubeconfig contents, SSH payloads, terminal contents, and transferred file contents.

### 5.4 Network Security

Requirements:

- Gateway Services shall be internal only.
- No `LoadBalancer`, `NodePort`, Ingress, public IP, VPN, or bastion path shall be required.
- The solution shall rely only on the Kubernetes `NetworkPolicy` API.
- NetworkPolicies shall restrict gateway Pod egress to approved development server addresses and TCP port 22.
- Private cloud firewalls should allow TCP 22 only from approved gateway source ranges.

### 5.5 Supply Chain Security

Requirements for later implementation phases:

- Container images shall be minimal and use distroless where practical.
- Images shall run as non-root.
- Filesystems should be read-only where practical.
- Linux capabilities shall be dropped unless explicitly required.
- SBOM generation shall be supported.
- Container signing shall be supported.
- Dependency scanning shall be part of CI/CD.

## 6. Maintainability

### 6.1 Code Maintainability

Requirements for later implementation phases:

- Go code shall be organized by clear package boundaries.
- Kubernetes, process management, filesystem, logging, and VS Code launcher concerns shall be separated.
- External command execution shall be abstracted for testability.
- Platform-specific behavior shall be isolated behind OS-specific implementations.
- Public behavior shall be driven by specifications and tests.

### 6.2 Configuration Maintainability

Requirements:

- Configuration shall be YAML.
- Configuration shall support multiple Kubernetes contexts, clusters, and gateways.
- Configuration shall remain compatible with future CRD/operator-managed inventory.
- Client configuration shall not become the long-term source of truth for backend server inventory.
- Configuration schema changes shall be versioned.

### 6.3 Operational Maintainability

Requirements:

- Operational ownership shall be split between platform operators and developers.
- Host keys shall be managed under the existing Platform GitOps repository in a dedicated `dev-connect` directory.
- Gateway changes shall follow the standard platform change process.
- Documentation shall include operational procedures for install, upgrade, rollback, troubleshooting, and key rotation.

## 7. Observability

### 7.1 Logging

Requirements:

- Logs shall be metadata-only.
- Logs shall include user identity where available, target server alias, session ID, start time, end time, duration, Kubernetes cluster/context, namespace, local port, gateway, and exit code.
- Logs shall not include terminal contents, SSH payloads, transferred file contents, passwords, tokens, private keys, or full kubeconfig content.
- `dev-connect` shall integrate with the enterprise logging platform already used by the Kubernetes platform.
- The first deployment logging backend shall be Azure Monitor / Log Analytics.
- Retention and forwarding to SIEM shall be handled by the logging backend, not by `dev-connect`.
- Development placeholder retention values are 30 days for metadata logs and 90 days for Kubernetes audit logs; they are configurable and not architectural constraints.

### 7.2 Metrics

Gateway metrics should include:

- Active TCP connections.
- Connection attempts.
- Connection failures.
- Tunnel startup duration.
- Active managed sessions.
- Reconnect attempts and failures.
- Gateway Pod CPU usage.
- Gateway Pod memory usage.
- Backend health.
- Backend connection duration.
- Bytes in/out where available without payload inspection.
- HAProxy process health.
- Pod restarts.

Client-side metrics or telemetry shall be opt-in unless enterprise policy requires managed endpoint telemetry.

Phase 4 shall define the minimum telemetry signals required for Kubernetes design and instrumentation. Concrete Azure Monitor / Log Analytics metric names and detailed dashboard layouts may be refined in later documentation, but the minimum required metrics shall be named no later than Phase 4.

### 7.3 Auditing

Requirements:

- Kubernetes audit logs shall capture port-forward requests where platform audit policy supports it.
- Audit correlation shall be possible across CLI session ID, Kubernetes audit records, gateway connection metadata, and target server SSH logs.
- Audit data shall remain metadata-only unless a future approved session-recording design is introduced outside the TCP-only gateway.

### 7.4 Alerting

The platform should alert on:

- No ready gateway Pods.
- Backend development server unreachable.
- High gateway connection failure rate.
- Repeated gateway Pod restarts.
- Kubernetes API server port-forward load approaching platform thresholds.
- Excessive reconnect failures.

## 8. Portability

### 8.1 Client Portability

The client shall support:

- Windows.
- Linux.
- macOS.

Requirements:

- OS-specific config paths shall be supported.
- OS-specific temporary file permission behavior shall be handled explicitly.
- VS Code launcher discovery shall check `PATH` first and then documented OS-specific default installation paths.
- OpenSSH behavior shall be compatible with VS Code Remote SSH.

### 8.2 Kubernetes Portability

Requirements:

- Kubernetes target version is v1.30+.
- The gateway design shall not require CNI-specific extensions.
- The gateway design shall rely only on standard Kubernetes `NetworkPolicy` API semantics.
- The design shall support Rancher-managed Kubernetes clusters.
- The design shall not depend on Azure-specific Kubernetes features, even though the first logging backend is Azure Monitor / Log Analytics.

### 8.3 Private Cloud Portability

Requirements:

- Development servers shall be addressable by stable IP or DNS.
- Ubuntu 24.04 LTS with standard OpenSSH is the first target server baseline.
- No assumption shall be made that development servers run inside Kubernetes.

## 9. Compatibility

### 9.1 External Tool Compatibility

Requirements:

- `kubectl` shall be detected automatically.
- The client shall support configured `kubectl` path override.
- Kubernetes communication shall be performed by invoking `kubectl`; direct Kubernetes API or Rancher API connections from `dev-connect` are forbidden.
- User-specific proxy overrides shall be supported for `kubectl` child processes without changing global proxy configuration.
- VS Code Desktop shall be launched through the command-line launcher.
- `--output json` shall be supported for machine-readable command responses and shall be independent from internal log format.
- `--log-format` shall control log formatting only.

### 9.2 Backward Compatibility

Requirements:

- Configuration schema versions shall be explicit.
- Deprecated fields shall be supported for at least one minor release after a replacement is introduced, unless security requires immediate removal.
- Breaking changes shall require a major version increment before general availability.

### 9.3 Forward Compatibility

Requirements:

- The first release shall not block future CRD/operator inventory.
- The first release shall not block dynamic gateway generation.
- The first release shall not block future multi-target routing.
- The first release shall not block multi-cluster support.

## 10. Recovery

### 10.1 Client Recovery

Requirements:

- The client shall detect stale session state.
- The client shall clean stale state safely.
- The client shall not kill unrelated `kubectl` processes.
- Reconnect shall be automatic by default.
- Reconnect shall use bounded exponential backoff.
- Reconnect shall stop after configured maximum attempts.
- Reconnect shall not retry SSH credentials.

### 10.2 Gateway Recovery

Requirements:

- Gateway Pods shall have readiness and liveness checks.
- Gateway Deployments shall support multiple replicas where useful.
- Pod disruption handling shall be defined in Phase 4.
- Gateway restart shall not require manual client reconfiguration.
- Initial gateway Pod sizing shall use fixed resource requests and limits.
- Initial request shall be 100 mCPU and 128 MiB memory per gateway Pod.
- Initial limit shall be 500 mCPU and 256 MiB memory per gateway Pod.
- Horizontal autoscaling may supplement the fixed initial sizing.
- Phase 7 performance tests shall measure actual utilization and adjust sizing where needed.

### 10.3 Session Timeout Recovery

Requirements:

- Idle timeout shall be configurable and default to 60 minutes.
- Maximum session duration shall be configurable and default to 12 hours.
- Temporary resources shall be cleaned gracefully after session end.
- Timeout cleanup shall not affect unrelated processes or user-managed SSH files.

## 11. Upgrade Strategy

### 11.1 Client Upgrade

Requirements:

- Client upgrades shall preserve compatible configuration files.
- The client shall report version, commit, build date, OS, and architecture.
- Upgrade notes shall document config schema changes, security changes, and behavior changes.
- Rollback shall be possible by installing the previous binary when config schema remains compatible.

### 11.2 Gateway Upgrade

Requirements:

- Gateway upgrades shall be compatible with Kubernetes rolling updates.
- HAProxy configuration changes shall be reviewable through GitOps.
- Host key inventory changes shall use the standard platform pull request workflow.
- Rollback shall be possible through GitOps revert where platform policy allows.

### 11.3 Compatibility During Upgrade

Requirements:

- Minor client versions should remain compatible with the same gateway API/config generation model.
- Gateway changes that require a new client behavior shall be coordinated and documented.
- Breaking upgrade paths shall require explicit release notes and migration guidance.

## 12. Versioning

### 12.1 Artifact Versions

Versioned artifacts shall include:

- `dev-connect` CLI.
- Gateway container image.
- Helm chart.
- Kustomize base/overlays.
- Configuration schema.
- Future CRDs.

### 12.2 Semantic Versioning

Requirements:

- The CLI, gateway image, and Helm chart should follow semantic versioning.
- Major versions may include breaking behavior or config schema changes.
- Minor versions may add backward-compatible features.
- Patch versions shall contain fixes without intended behavior changes.

### 12.3 API and Schema Versioning

Requirements:

- YAML configuration shall include `apiVersion`.
- Future CRDs shall use Kubernetes API versioning conventions.
- Version migration guidance shall be documented.

## 13. Acceptance Criteria

Phase 3 is acceptable when:

- Performance requirements define capacity, throughput, latency targets, and Kubernetes API load considerations.
- Security requirements cover identity, SSH, secrets, network security, and supply chain expectations.
- Maintainability requirements cover code, configuration, and operations.
- Observability requirements cover logs, metrics, auditing, and alerting.
- Portability requirements cover OS, Kubernetes, CNI, Rancher, and private cloud portability.
- Compatibility requirements cover tools, config, backward compatibility, and future compatibility.
- Recovery requirements cover client, gateway, reconnect, and timeout cleanup.
- Upgrade strategy covers client, gateway, GitOps, rollback, and compatibility.
- Versioning requirements cover binaries, images, charts, schema, and future CRDs.

## 14. Phase Gate

This document completes the approved Phase 3 non-functional requirements. Phase 4 may start.
