# dev-connect Kubernetes Design

Status: Approved

## 1. Purpose

This document defines the Kubernetes design for the `dev-connect` gateway.

Phase 4 builds on the approved Phase 1 architecture, Phase 2 functional specification, and Phase 3 non-functional requirements. It defines the intended Kubernetes resources and packaging model but does not create implementation manifests, Helm chart files, Kustomize files, container images, Go code, or tests.

## 2. Design Scope

In scope:

- Namespace.
- HAProxy gateway Deployment.
- ClusterIP Service.
- ConfigMap.
- NetworkPolicy.
- PodDisruptionBudget.
- HorizontalPodAutoscaler.
- RBAC.
- Kustomize packaging design.
- Helm chart design.
- Minimum telemetry signals.

Out of scope:

- Actual Kubernetes YAML files.
- Helm template implementation.
- Kustomize implementation.
- Gateway container build.
- Go client implementation.
- Automated tests.

## 3. Phase Inputs

Approved constraints:

- Kubernetes v1.30+.
- Gateway remains TCP-only.
- Gateway does not perform SSH authentication.
- SSH authentication remains on the remote Linux development server.
- No SSH private keys, passwords, Kubernetes tokens, or user database in Kubernetes.
- No public endpoint, VPN, bastion host, `LoadBalancer`, `NodePort`, or Ingress.
- Kubernetes authentication and authorization follow the existing Rancher authentication model.
- Access is granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.
- `dev-connect` never establishes direct network connections to Rancher or the Kubernetes API.
- All Kubernetes communication is delegated to the locally installed `kubectl` binary using the user's existing kubeconfig and enterprise proxy configuration.
- Optional user-specific proxy overrides apply only to `kubectl` processes started by `dev-connect`.
- Gateway uses HAProxy.
- First-release gateway model is one HAProxy TCP listener on port 22 with one backend target per gateway Deployment.
- Multi-target routing is deferred to a later phase using separate gateway Deployments per target or dynamic gateway generation.
- Network security relies only on the Kubernetes `NetworkPolicy` API.
- Target development servers are addressed by stable IP or DNS.
- Initial gateway Pod request is 100 mCPU and 128 MiB memory.
- Initial gateway Pod limit is 500 mCPU and 256 MiB memory.

## 4. Resource Model

For the first release, each target development server is represented by one gateway Deployment and one Gateway Service.

Example target mapping:

| Target | Gateway Deployment | Gateway Service | Backend |
| --- | --- | --- | --- |
| `dev01` | `dev-connect-gateway-dev01` | `dev-connect-gateway-dev01` | `dev01.example.internal:22` |
| `dev02` | `dev-connect-gateway-dev02` | `dev-connect-gateway-dev02` | `10.20.30.42:22` |

Rationale:

- Keeps HAProxy TCP configuration simple.
- Avoids SSH metadata inspection.
- Avoids SSH termination.
- Avoids credential-bearing routing.
- Preserves future compatibility with operator-generated gateways.

## 5. Namespace Design

Namespace:

- Name: `dev-connect`.
- Managed by platform GitOps.
- Contains gateway Deployments, Services, ConfigMaps, NetworkPolicies, PDBs, HPAs, and namespaced RBAC.

Requirements:

- Namespace labels shall support ownership, environment, and policy selection.
- Namespace shall not contain SSH private keys, user passwords, or user databases.
- Namespace shall be compatible with Rancher-managed access controls.

Recommended labels:

```yaml
app.kubernetes.io/name: dev-connect
app.kubernetes.io/part-of: dev-connect
app.kubernetes.io/managed-by: platform-gitops
```

## 6. Deployment Design

Workload:

- Kubernetes `Deployment`.
- One Deployment per backend target in the first release.
- Container: HAProxy.
- Container listener: non-privileged TCP port `2222`.
- Kubernetes Service port: standard SSH TCP port `22`.
- Backend: exactly one development server endpoint on TCP port 22.

Deployment naming:

```text
dev-connect-gateway-<target>
```

Container ports:

- `ssh-tcp`: 2222/TCP.
- `metrics`: optional metrics port for HAProxy exporter or HAProxy stats endpoint.

Pod security requirements:

- Run as non-root.
- HAProxy shall listen on non-privileged container port `2222`.
- The Kubernetes Service shall expose port `22` and forward to target port `2222`.
- The container shall not require root or `NET_BIND_SERVICE` for SSH forwarding.
- Drop Linux capabilities unless explicitly required.
- Use read-only root filesystem where practical.
- Disable privilege escalation.
- Use seccomp `RuntimeDefault`.

Initial resources:

```yaml
requests:
  cpu: 100m
  memory: 128Mi
limits:
  cpu: 500m
  memory: 256Mi
```

Availability:

- Replica count shall be environment-specific and configurable through Helm values or Kustomize overlays.
- Recommended development replicas: 1.
- Recommended test/integration replicas: 2.
- Recommended production replicas: 2.
- Topology spread or anti-affinity should spread gateway Pods across nodes where practical.
- Rolling updates shall preserve availability where possible.

Health checks:

- Readiness probe shall verify HAProxy is accepting TCP connections or expose an HAProxy health endpoint.
- Liveness probe shall detect stuck HAProxy process.
- Startup probe may be used when HAProxy config generation or validation needs additional startup time.

## 7. HAProxy Configuration Design

Configuration source:

- Kubernetes `ConfigMap`.
- One ConfigMap per gateway target or one generated ConfigMap per Deployment.
- Managed by GitOps for the first release.
- Compatible with future operator-generated configuration.

Required behavior:

- Listen on one TCP frontend.
- Forward to exactly one backend target.
- Do not terminate SSH.
- Do not inspect credentials.
- Do not log payloads.
- Emit metadata-only connection logs.
- Enable backend health checks where safe and supported.

Conceptual HAProxy structure:

```text
global
  log stdout format raw local0

defaults
  mode tcp
  timeout connect 10s
  timeout client 12h
  timeout server 12h
  option tcplog

frontend ssh
  bind :2222
  default_backend target

backend target
  server dev01 dev01.example.internal:22 check
```

Notes:

- The container shall listen on `2222` while the Service exposes port `22`.
- Timeouts shall align with Phase 3 session timeout defaults where practical.
- Backend health checks must not authenticate to SSH.

## 8. Service Design

Service type:

- `ClusterIP`.

Naming:

```text
dev-connect-gateway-<target>
```

Ports:

- Service port: `22`.
- Target port: HAProxy container listener `2222`.

Requirements:

- No `LoadBalancer`.
- No `NodePort`.
- No Ingress.
- Used as the target for `kubectl port-forward service/<service-name> <localPort>:22`.
- Service selector shall match exactly one gateway Deployment target group.

## 9. NetworkPolicy Design

NetworkPolicies shall use only the Kubernetes `NetworkPolicy` API.

Required policies:

1. Default deny ingress and egress for the `dev-connect` namespace.
2. Allow required ingress to gateway Pods for Kubernetes port-forward traffic where the CNI can express it.
3. Allow gateway Pod egress to the approved backend development server IP or DNS-resolved IPs on TCP port 22.
4. Allow DNS egress only when DNS-based backend addressing is configured.
5. Allow metrics scraping from approved monitoring namespace or scraper Pods.

Constraints:

- Do not rely on CNI-specific policy extensions in the first release.
- Private cloud firewalls should also restrict TCP 22 access to approved gateway source ranges.
- First implementation shall support both DNS names and IP addresses for backend targets.
- DNS names are the recommended and preferred backend addressing format.
- IP addresses are supported for environments without internal DNS.
- DNS-based backends require DNS egress and a reliable operational process for keeping NetworkPolicy egress targets aligned with resolved IPs, because standard NetworkPolicy is IP/CIDR based for external destinations.
- Static IP based deployments shall not be granted DNS egress unless required for another approved reason.

## 10. PodDisruptionBudget Design

PDB:

- One PDB per gateway Deployment or per app selector group.
- Production default: `minAvailable: 1`.

Requirements:

- Planned node maintenance should not evict all gateway Pods for a target at once.
- Existing port-forward sessions may still break when a selected Pod is disrupted.
- Client automatic reconnect handles transient tunnel interruption.

## 11. HorizontalPodAutoscaler Design

HPA:

- One HPA per gateway Deployment where metrics are available.
- HPA shall be disabled by default in the first implementation.
- The design shall support enabling HPA later without redesign.
- When enabled, min replicas and max replicas shall be configurable.
- Recommended initial max replicas when HPA is enabled later: 5.

Initial scaling signals:

- CPU utilization.
- Memory utilization.

Future scaling signals:

- Active TCP connections.
- Connection rate.
- HAProxy queue or failure metrics.

Notes:

- Static replicas simplify troubleshooting and capacity planning during the first rollout.
- Phase 7 performance tests shall provide sizing data before enabling HPA by default.
- HPA does not preserve existing port-forward streams during Pod termination.
- Rolling updates and disruption settings must account for session interruption and client reconnect behavior.

## 12. RBAC Design

### 12.1 Developer RBAC

Developer access shall be granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.

Developer permissions shall be scoped to the `dev-connect` namespace.

`dev-connect` shall exercise these permissions only through locally invoked `kubectl` commands. The client shall not use a direct Kubernetes API or Rancher API client.

Required access:

- Read gateway discovery resources:
  - `services`: `get`, `list`.
  - selected `pods`: `get`, `list`.
  - endpoint resources required by the supported `kubectl port-forward` behavior: `get`, `list`.
- Create port-forward sessions:
  - `pods/portforward`: `create`.
- Optional preflight:
  - `selfsubjectaccessreviews.authorization.k8s.io`: `create`, where required and permitted by cluster policy.

Developer access shall not include:

- read Secrets,
- create Pods,
- update ConfigMaps,
- exec into Pods,
- update Deployments,
- access unrelated namespaces.

Recommended group:

- `dev-connect-users`.

### 12.2 Operator RBAC

Operator access shall support GitOps-managed deployment and operations.

Recommended groups:

- `dev-connect-admins`.
- `platform-admins`.

Operator permissions may include managing:

- Deployments.
- Services.
- ConfigMaps.
- NetworkPolicies.
- PodDisruptionBudgets.
- HorizontalPodAutoscalers.
- ServiceAccounts.
- Roles.
- RoleBindings.

Operators shall not manage SSH user credentials in Kubernetes because SSH authentication remains on the development server.

## 13. ServiceAccount Design

Gateway Pods shall use a dedicated ServiceAccount:

```text
dev-connect-gateway
```

Requirements:

- `automountServiceAccountToken` shall be disabled by default.
- Token automount shall only be enabled for workloads that explicitly require Kubernetes API access.
- Gateway Pods should not need Kubernetes API access at runtime in the first release.
- No Secret read permissions are required for the gateway runtime.
- Gateway runtime does not participate in client Kubernetes authentication; the local `kubectl` process handles Kubernetes communication from the developer workstation.
- Any future Operator or controller requiring Kubernetes API access shall use a dedicated ServiceAccount with explicitly enabled token mounting.

## 14. Telemetry Design

Phase 4 minimum telemetry signals:

- Tunnel startup duration, emitted by the client and correlated with session ID.
- Active sessions, from client state and/or gateway connection metrics.
- Gateway connection attempts.
- Gateway connection failures.
- Backend health state.
- HAProxy process health.
- Gateway Pod CPU usage.
- Gateway Pod memory usage.
- Pod restarts.
- Reconnect attempts and failures.

First deployment backend:

- Azure Monitor / Log Analytics.

Requirements:

- Gateway logs shall be metadata-only.
- Logs shall not include SSH payloads, terminal contents, file contents, credentials, tokens, private keys, or kubeconfig contents.
- Retention and forwarding to SIEM are handled by the logging backend, not by `dev-connect`.
- Dashboard layouts and exact Azure metric names may be refined in later documentation.

## 15. Kustomize Design

Kustomize shall be supported for platform GitOps workflows.

Recommended structure for later implementation:

```text
kubernetes/
  base/
    namespace.yaml
    serviceaccount.yaml
    role.yaml
    rolebinding.yaml
    deployment.yaml
    service.yaml
    configmap.yaml
    networkpolicy.yaml
    pdb.yaml
    hpa.yaml
    kustomization.yaml
  overlays/
    dev/
      kustomization.yaml
      patch-target-dev01.yaml
    prod/
      kustomization.yaml
      patch-target-dev01.yaml
```

Design requirements:

- Base shall contain reusable resource templates.
- Overlays shall define target-specific backend host/IP and environment-specific replicas/resources.
- Secrets shall not be used for SSH keys or credentials.
- Host public keys may be referenced as non-secret configuration where required by client inventory, not by the gateway for authentication.

## 16. Helm Chart Design

Helm shall be supported for environments that prefer chart-based installation.

Recommended chart:

```text
charts/dev-connect-gateway/
  Chart.yaml
  values.yaml
  templates/
    namespace.yaml
    serviceaccount.yaml
    role.yaml
    rolebinding.yaml
    deployment.yaml
    service.yaml
    configmap.yaml
    networkpolicy.yaml
    pdb.yaml
    hpa.yaml
```

Required values:

```yaml
target:
  name: dev01
  backendHost: dev01.example.internal
  backendPort: 22

service:
  port: 22
  targetPort: 2222

replicaCount: 2
# Recommended defaults:
# development: 1
# test/integration: 2
# production: 2

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi

autoscaling:
  enabled: false
  minReplicas: 2
  maxReplicas: 5

networkPolicy:
  enabled: true
  dnsEgressEnabled: true
```

Design requirements:

- Helm chart shall not template SSH credentials.
- Helm chart shall not create public endpoints.
- Helm chart shall support disabling Namespace creation for GitOps-managed namespaces.
- Helm chart shall support metadata labels required by the platform.
- Helm chart shall support both DNS and IP backend addressing.
- Helm chart shall enable DNS egress only when DNS-based backend addressing is configured.
- Helm chart shall disable ServiceAccount token automount by default.
- Helm chart shall support `helm lint` and template rendering tests in later phases.

## 17. Upgrade and Rollback Design

Requirements:

- Gateway changes shall be GitOps-managed.
- Rollback shall use GitOps revert where platform process allows.
- ConfigMap changes shall roll Pods when HAProxy config changes.
- Rolling updates shall preserve at least one available Pod where replicas allow.
- Client reconnect handles interrupted port-forward sessions.

## 18. Security Review Points

Phase 4 implementation shall verify:

- No public Service type.
- No Ingress.
- No NodePort.
- No SSH credentials in Kubernetes.
- Gateway ServiceAccount has no unnecessary API permissions.
- Developer RBAC has no Secret read, exec, Pod create, ConfigMap update, or unrelated namespace access.
- NetworkPolicies restrict egress to approved backend TCP 22 destinations.
- HAProxy does not terminate SSH.
- HAProxy does not log payloads.

## 19. Acceptance Criteria

Phase 4 is acceptable when:

- Kubernetes resource design covers Namespace, Deployment, Service, ConfigMap, NetworkPolicy, PDB, HPA, RBAC, ServiceAccount, Kustomize, and Helm.
- The first-release one-backend-per-gateway-Deployment model is clear.
- The design preserves TCP-only forwarding and server-side SSH authentication.
- The design avoids public endpoints and credentials in Kubernetes.
- Developer and operator RBAC are clearly separated.
- NetworkPolicy constraints are clear and CNI-independent.
- Initial resource requests and limits are specified.
- Minimum telemetry signals are defined.
- Kustomize and Helm packaging models are specified.

## 20. Phase Gate

This document completes the approved Phase 4 Kubernetes design. Phase 5 may start.
