# dev-connect Phase 1 Review

Status: Approved

## Scope

Phase 1 covers architecture only. It intentionally does not include implementation code, Kubernetes manifests, Helm chart files, tests, or CI/CD definitions.

Primary artifact:

- `architecture.md`

## Review Checklist

Architecture:

- [x] Overall architecture is understandable and complete.
- [x] Component responsibilities are clear.
- [x] Communication flows satisfy the constraint that SSH traffic traverses the Kubernetes API.
- [x] Gateway role is limited to TCP forwarding.
- [x] SSH authentication remains on the remote Linux development server.
- [x] VS Code Desktop and local GitHub Copilot behavior are preserved.
- [x] No VPN, bastion host, public IP, browser-based VS Code, or development workload inside Kubernetes is introduced.

Security:

- [x] Kubernetes authentication and authorization model is acceptable.
- [x] Least-privilege RBAC assumptions are acceptable.
- [x] No credentials, SSH keys, or user database are stored in Kubernetes.
- [x] NetworkPolicy and private firewall expectations are acceptable.
- [x] Threat model covers the main enterprise risks.

Operations:

- [x] Failure handling is acceptable for first release.
- [x] Logging and audit expectations are acceptable.
- [x] Monitoring signals are sufficient for production operations.
- [x] Scalability and Kubernetes API server capacity risks are understood.
- [x] High availability limitations and reconnect expectations are acceptable.

Extensibility:

- [x] Future dynamic gateway configuration is not blocked.
- [x] Multiple development servers are supported by the architecture.
- [x] Multi-cluster and multiple private cloud support are not blocked.
- [x] Future service discovery, auditing, Rancher-backed enterprise identity integration, and operator management are not blocked.

## Recorded Decisions

The following Phase 1 decisions are recorded and should be used by the functional and Kubernetes specifications:

1. First-release gateway routing model:
   - one HAProxy TCP listener on port 22.
   - one backend target per gateway Deployment.
   - multi-target routing is deferred to a later phase using separate gateway Deployments per target or dynamic gateway generation.
2. Source of server inventory:
   - future CRD/operator control plane.
   - first implementation artifacts must remain compatible with this model and must not make local client YAML the long-term source of truth.
3. Required Kubernetes RBAC verbs and target resources for the supported `kubectl port-forward` mode.
   - read gateway discovery resources in the gateway namespace, including `services`, selected `pods`, and endpoint resources required by the chosen `kubectl` behavior.
   - `create` on `pods/portforward` in the gateway namespace.
   - optional `create` on `selfsubjectaccessreviews.authorization.k8s.io` for explicit preflight checks where required by cluster policy.
   - no permission to read Secrets, exec into Pods, create Pods, update ConfigMaps, or access unrelated namespaces.
4. Development server addressing model:
   - stable IP.
   - DNS.
5. Reconnect policy:
   - automatic by default.
6. Concurrent developer sizing:
   - initial production design target is 100 simultaneous developers.
   - growth target is 250 simultaneous developers without architectural redesign.
   - default client behavior is at most one active port-forward per developer.
7. Identity, authentication, and RBAC:
   - Kubernetes authentication and authorization follow the existing Rancher authentication model.
   - the solution introduces no separate identity store.
   - access permissions are granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.
   - local Kubernetes users are not part of the dev-connect access model.
   - RBAC is assigned through groups such as `dev-connect-users`, `dev-connect-admins`, and `platform-admins`.
   - least privilege for developers.
8. CNI and NetworkPolicy:
   - architecture remains CNI-independent.
   - the solution relies only on the Kubernetes `NetworkPolicy` API.
   - CNI-specific extensions are not used unless explicitly justified in a later phase.
9. SSH host key policy:
   - host key pinning is mandatory.
   - `StrictHostKeyChecking=no` is forbidden.
   - host keys are centrally managed.
   - the dev-connect client loads the expected host key at connection startup and writes it to a temporary `known_hosts` file.
   - host keys are managed as infrastructure configuration through the existing GitOps process.
   - pinned SSH host keys are owned by the existing Platform GitOps repository under a dedicated `dev-connect` directory.
   - changes require the standard platform pull request approval workflow.
   - rotation follows a controlled GitOps flow: host key generation, GitOps change, deployment, and client use of the new key.
10. Logging and auditing:
   - metadata only.
   - record user, target server, start time, end time, duration, Kubernetes context, local port, and exit code.
   - do not record terminal contents, SSH payloads, transferred file contents, passwords, tokens, or SSH private keys.
11. Kubernetes API load:
   - maximum one port-forward per developer by default.
   - gateway is horizontally scalable.
   - API server load from port-forward must be monitored, especially as usage grows beyond a few dozen users.
12. Log retention:
   - retention is configurable.
   - log retention is not enforced by the application.
   - retention is enforced by the centralized enterprise logging backend according to enterprise retention policies.
   - the target platform logging backend for the first deployment is Azure Monitor / Log Analytics.
   - retention and forwarding to SIEM are handled by the logging backend, not by dev-connect.
   - development placeholder values are 30 days for dev-connect metadata logs and 90 days for Kubernetes audit logs.
   - development placeholder values are not architectural constraints.
13. Performance test throughput profile:
   - average throughput per developer is 0.5 Mbit/s.
   - peak throughput per developer is 25 Mbit/s.
   - activity-specific peaks include Git operations and file synchronization bursts.
14. Session timeouts:
   - idle timeout is configurable and defaults to 60 minutes.
   - maximum session duration is configurable and defaults to 12 hours.
   - automatic reconnect remains enabled for short `kubectl port-forward` interruptions.
   - temporary resources are cleaned up gracefully after session end.
15. Kubernetes communication and proxy handling:
   - `dev-connect` shall never establish direct network connections to Rancher or the Kubernetes API.
   - all Kubernetes communication shall be delegated to the locally installed `kubectl` binary.
   - `kubectl` shall use the user's existing kubeconfig and enterprise proxy configuration by default.
   - optional user-specific proxy overrides are supported when required.
   - proxy overrides apply only to `kubectl` processes started by `dev-connect`.
   - proxy overrides shall not modify the user's global operating system proxy settings, corporate proxy configuration, kubeconfig, or shell profile.

## Remaining Implementation Decisions

No architecture-level questions remain open.

## Approval Gate

Phase 1 is approved. Phase 2 has also been approved by stakeholder decision, so Phase 3 may start.
