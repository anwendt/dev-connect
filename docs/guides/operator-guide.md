# dev-connect Operator Guide

Status: Implementation guide

## Purpose

This guide describes the operational responsibilities for running `dev-connect` gateways on Kubernetes.

For step-by-step operational procedures, use the
[Operations Runbook](../runbooks/operations-runbook.md).

## Operator Responsibilities

Platform operators manage:

- `dev-connect` namespace,
- HAProxy gateway Deployments,
- ClusterIP Services,
- ConfigMaps,
- NetworkPolicies,
- PodDisruptionBudgets,
- optional HPAs,
- Rancher-managed RBAC,
- GitOps-owned SSH host key inventory,
- logging and monitoring integration.

Operators do not manage SSH private keys, user passwords, or SSH user databases in Kubernetes.

## Gateway Operating Model

First release:

- one gateway Deployment per target development server,
- one ClusterIP Service per gateway Deployment,
- HAProxy listens on container port `2222`,
- Service exposes TCP port `22`,
- HAProxy forwards to one backend development server on TCP `22`.

Recommended replicas:

| Environment | Replicas |
| --- | --- |
| Development | 1 |
| Test / Integration | 2 |
| Production | 2 |

HPA is disabled by default for the first implementation. The architecture supports enabling HPA after performance data is available.

## Resource Defaults

Initial gateway Pod resources:

```yaml
requests:
  cpu: 100m
  memory: 128Mi
limits:
  cpu: 500m
  memory: 256Mi
```

These values shall be validated through Phase 7 performance tests.

## Health Probes

Gateway Pods expose HAProxy on container port `2222`.

The Kubernetes Deployment uses TCP probes:

- readiness probe on port `2222`,
- liveness probe on port `2222`.

The Service continues to expose standard SSH port `22` and forwards to container port `2222`.

## NetworkPolicy

The namespace should follow default deny ingress and egress.

Gateway egress shall be restricted to:

- approved target development server addresses,
- TCP port `22`,
- DNS only when DNS-based backend addressing is configured,
- monitoring endpoints only when required.

No `LoadBalancer`, `NodePort`, or Ingress is allowed.

## RBAC

Developer access shall be least privilege and namespace-scoped.

Expected developer access:

- read gateway discovery resources,
- create `pods/portforward`,
- optional self access review where permitted.

Forbidden developer access:

- Secrets,
- Pod creation,
- Pod exec,
- Deployment modification,
- ConfigMap writes,
- unrelated namespaces,
- cluster-admin permissions.

## Logging and Monitoring

Logs contain metadata only.

HAProxy is configured for TCP logging to stdout:

- `mode tcp`,
- `log stdout format raw local0`,
- `option tcplog`,
- no SSH payload logging,
- no SSH termination.

Recommended operational signals:

- active sessions,
- tunnel establishment failures,
- gateway Pod readiness,
- HAProxy backend health,
- gateway CPU and memory,
- Kubernetes API port-forward error rates,
- reconnect counts,
- session duration,
- exit code distribution.

First deployment logging backend: Azure Monitor / Log Analytics.

Retention is enforced by the enterprise logging platform, not by `dev-connect`.

Documentation shall reference logical dashboard names rather than platform-specific URLs.

Recommended logical dashboards:

- `dev-connect Operations`
- `dev-connect Gateway`
- `dev-connect Security`

Platform-specific dashboard links shall be provided by the deployment documentation of the target environment.

## Production Rollout Checklist

Before production rollout, operators must validate:

- Rancher-managed group membership for `dev-connect-users`.
- Namespace-scoped RBAC grants only the approved resources and verbs.
- `kubectl auth can-i create pods/portforward` succeeds for authorized users.
- A functional `kubectl port-forward` preflight succeeds through Rancher.
- NetworkPolicy allows only Kubernetes API ingress to the Service path and egress to approved backend SSH targets.
- DNS egress is enabled only for DNS-based backend addressing.
- Gateway Pods become ready and HAProxy backend health checks pass.
- Logs arrive in Azure Monitor / Log Analytics or the target enterprise logging backend.
- Host key inventory is managed by Platform GitOps with two-admin approval.
- Windows, Linux, and macOS clients can run `dev-connect config validate`, `list`, `connect --no-code`, `status`, and `disconnect` in a controlled smoke test.

The formal production load test is a separate pre-rollout activity and is not included in the current implementation slice.

## Host Key Operations

Pinned SSH host keys are infrastructure configuration.

Production host key inventory shall be managed through the enterprise Platform GitOps repository. The exact repository name is intentionally not fixed.

Recommended logical structures:

```text
platform/
  dev-connect/
    hostkeys/
      known_hosts
```

or:

```text
clusters/
  production/
    dev-connect/
      hostkeys.yaml
```

The concrete repository path shall be defined by the Platform Engineering team during production onboarding.

Host key changes:

1. Generate new target host key.
2. Update the Platform GitOps repository under the dedicated `dev-connect` directory.
3. Require pull request approval by at least two authorized platform administrators.
4. Deploy through the normal GitOps process.
5. Clients use the updated pinned key on the next connection.
