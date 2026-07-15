# dev-connect Operations Runbook

Status: Approved

## Purpose

This runbook describes how platform operators install, validate, operate,
troubleshoot, roll back, and recover `dev-connect` gateway deployments.

It assumes:

- Rancher-managed Kubernetes access,
- no direct client connection to Rancher or the Kubernetes API except through `kubectl`,
- HAProxy gateway deployed through Helm or Kustomize,
- private development servers outside Kubernetes,
- SSH authentication performed only on the target Linux development server,
- metadata-only logging,
- Rancher Monitoring or a compatible Prometheus Operator stack.

## Operator Scope

Operators are responsible for:

- namespace lifecycle,
- gateway deployments,
- Service, NetworkPolicy, RBAC, PDB, and optional HPA resources,
- Rancher RBAC group assignments,
- GitOps-managed pinned SSH host key inventory,
- monitoring, alerts, dashboards, and logging integration,
- release rollout and rollback,
- incident triage.

Operators are not responsible for storing SSH private keys, SSH passwords,
Kubernetes credentials, or user databases in Kubernetes.

## Production Readiness Checklist

Complete this checklist before production rollout.

| Area | Check |
| --- | --- |
| Identity | Access is granted through existing Rancher authentication and enterprise identity provider integration. |
| RBAC | Developer permissions are namespace-scoped and least privilege. |
| Host keys | Production SSH host keys are pinned and managed through Platform GitOps. |
| Network | No `LoadBalancer`, `NodePort`, Ingress, VPN, bastion, or public IP path exists. |
| Gateway | Gateway Pods are ready and expose HAProxy on non-privileged port `2222`. |
| Service | Kubernetes Service exposes TCP `22` and targets container port `2222`. |
| NetworkPolicy | Egress is limited to approved development server SSH targets and DNS only when required. |
| Monitoring | ServiceMonitor, PrometheusRule, and Grafana dashboard are deployed where Rancher Monitoring can discover them. |
| Logging | Metadata logs are forwarded to the enterprise logging backend. |
| Client | Windows, macOS, and Linux smoke tests pass. |
| Documentation | Installation, configuration, troubleshooting, and host key rotation procedures are published. |

The formal production load test remains a separate pre-production activity.

## Deployment Procedure

### 1. Confirm Cluster Access

```text
export KUBECONFIG=/path/to/cluster.yaml
kubectl cluster-info
kubectl auth can-i get pods -n dev-connect
```

Use the kubeconfig and Rancher context approved for the target platform
environment.

### 2. Verify Monitoring CRDs

```text
kubectl get crd servicemonitors.monitoring.coreos.com
kubectl get crd prometheusrules.monitoring.coreos.com
kubectl get namespace cattle-monitoring-system
kubectl get namespace cattle-dashboards
```

If Rancher Monitoring is not installed, deploy the gateway without monitoring
resources or install the platform monitoring stack first.

### 3. Install or Upgrade Gateway

Example for target `dev01`:

```text
helm upgrade --install dev-connect-dev01 charts/dev-connect-gateway \
  --namespace dev-connect \
  --create-namespace \
  --set target.name=dev01 \
  --set target.host=172.28.192.14 \
  --set target.port=22 \
  --set networkPolicy.backendCIDR=172.28.192.14/32 \
  --set networkPolicy.dnsEgress.enabled=false \
  --set monitoring.enabled=true \
  --wait \
  --timeout 5m
```

Use DNS-based backend addressing only when internal DNS is available and DNS
egress is explicitly allowed by NetworkPolicy.

### 4. Verify Rollout

```text
kubectl rollout status deployment/dev-connect-gateway-dev01 -n dev-connect
kubectl get pods -n dev-connect -l app.kubernetes.io/instance=dev-connect-dev01
kubectl get svc -n dev-connect dev-connect-gateway-dev01
kubectl get endpoints -n dev-connect dev-connect-gateway-dev01
```

Expected result:

- deployment rollout succeeds,
- Pods are `Ready`,
- Service is `ClusterIP`,
- Service exposes `22/TCP` and metrics port `8404/TCP`,
- no public endpoint is created.

### 5. Verify Monitoring Resources

```text
kubectl get servicemonitor -n dev-connect
kubectl get prometheusrule -n dev-connect
kubectl get configmap -n cattle-dashboards dev-connect-gateway-dev01-dashboard
```

The dashboard is named:

```text
dev-connect Gateway
```

The server filter is:

```text
Server
```

The underlying Prometheus label is:

```text
dev_connect_target
```

For the current deployment, the value should be:

```text
dev01
```

Grafana and Prometheus may require a short scrape and sidecar sync delay before
the dashboard and target filter appear.

### 6. Verify Developer RBAC

Run checks as an authorized developer identity:

```text
kubectl auth can-i get services -n dev-connect
kubectl auth can-i get pods -n dev-connect
kubectl auth can-i create pods/portforward -n dev-connect
```

Forbidden checks should remain denied:

```text
kubectl auth can-i get secrets -n dev-connect
kubectl auth can-i delete pods -n dev-connect
kubectl auth can-i update deployments -n dev-connect
kubectl auth can-i create clusterrolebindings
```

### 7. Run Client Smoke Test

Use a production-like client configuration with approved pinned host keys.

```text
dev-connect --config /path/to/config.yaml config validate
dev-connect --config /path/to/config.yaml list
dev-connect --config /path/to/config.yaml connect dev01 --no-code --output json
dev-connect status --output json
dev-connect disconnect --output json
```

Then run one full VS Code smoke test:

```text
dev-connect --config /path/to/config.yaml connect dev01
```

Expected result:

- `kubectl` preflight succeeds,
- temporary SSH config and `known_hosts` are generated,
- `kubectl port-forward` stays running,
- VS Code Desktop opens Remote SSH target `dev01`,
- VS Code Server installs on the target server when needed,
- GitHub Copilot remains local in VS Code Desktop,
- `disconnect` cleans up local temporary resources.

## Day-2 Operations

### Routine Health Checks

Daily or after platform maintenance:

```text
kubectl get deploy,pod,svc,pdb,networkpolicy -n dev-connect
kubectl get servicemonitor,prometheusrule -n dev-connect
kubectl get configmap -n cattle-dashboards -l grafana_dashboard=1
```

Check Rancher or Grafana dashboards:

- `dev-connect Gateway`
- `dev-connect Operations`
- `dev-connect Security`

Minimum signals:

- gateway Pod readiness,
- HAProxy backend availability,
- active/current connections,
- connection errors,
- traffic in/out,
- Pod CPU and memory,
- restart count,
- Kubernetes API port-forward failures,
- client metadata error rates.

### Metadata Log Review

Logs must contain metadata only:

- user,
- target server,
- session ID,
- start and end time,
- duration,
- exit code,
- Kubernetes cluster,
- namespace,
- local port.

Logs must not contain:

- SSH payload,
- terminal content,
- passwords,
- SSH private keys,
- Kubernetes tokens,
- bearer tokens,
- proxy credentials,
- kubeconfig credentials.

Retention is enforced by the enterprise logging backend, not by `dev-connect`.

## Host Key Rotation

Use this process when a development server host key is intentionally changed.

1. Generate or replace the SSH host key on the target development server.
2. Record the new public host key.
3. Update the Platform GitOps host key inventory under the approved
   `dev-connect` path.
4. Open a pull request.
5. Require approval by at least two authorized platform administrators.
6. Deploy through the normal GitOps process.
7. Validate the affected client configuration:

```text
dev-connect --config /path/to/config.yaml config validate
```

8. Run a smoke connection:

```text
dev-connect --config /path/to/config.yaml connect dev01 --no-code
dev-connect disconnect
```

Do not use:

```text
StrictHostKeyChecking=no
```

Unknown or mismatched host keys must fail closed.

## Rollback

### Helm Rollback

List release revisions:

```text
helm history dev-connect-dev01 -n dev-connect
```

Rollback to the last known good revision:

```text
helm rollback dev-connect-dev01 <revision> -n dev-connect --wait --timeout 5m
```

Verify:

```text
kubectl rollout status deployment/dev-connect-gateway-dev01 -n dev-connect
kubectl get pods -n dev-connect -l app.kubernetes.io/instance=dev-connect-dev01
```

### Client Rollback

Client binaries are versioned release artifacts. Roll back by reinstalling the
last approved release version and verifying:

```text
dev-connect version
dev-connect config validate
dev-connect connect dev01 --no-code
dev-connect disconnect
```

## Incident Response

### Gateway Down

Symptoms:

- dashboard shows backend unavailable,
- gateway Pods not ready,
- client tunnel preflight fails.

Checks:

```text
kubectl get pods -n dev-connect
kubectl describe pod -n dev-connect -l app.kubernetes.io/instance=dev-connect-dev01
kubectl logs -n dev-connect deployment/dev-connect-gateway-dev01
kubectl get events -n dev-connect --sort-by=.lastTimestamp
```

Likely causes:

- backend development server unreachable,
- NetworkPolicy denies egress,
- invalid HAProxy configuration,
- node disruption,
- resource pressure.

### RBAC Failure

Symptoms:

- `kubectl auth can-i` fails,
- `dev-connect connect` fails during preflight,
- users report authorization errors after Rancher group changes.

Checks:

```text
kubectl auth can-i create pods/portforward -n dev-connect --as <user-or-group>
kubectl get role,rolebinding -n dev-connect
```

Action:

- verify Rancher-managed group membership,
- verify RoleBinding subjects,
- do not grant cluster-admin,
- do not grant Secrets access.

### Port-Forward Instability

Symptoms:

- session starts and then drops,
- reconnect attempts increase,
- Kubernetes API server shows elevated streaming load.

Checks:

```text
kubectl get pods -n dev-connect
kubectl top pods -n dev-connect
kubectl get events -n dev-connect --sort-by=.lastTimestamp
```

Action:

- check Kubernetes API server health,
- check enterprise proxy behavior for streaming connections,
- inspect gateway Pod restarts,
- compare current usage with expected concurrent developer count,
- run targeted performance diagnostics if the issue repeats.

### Dashboard Missing

Symptoms:

- `dev-connect Gateway` dashboard does not appear in Rancher/Grafana.

Checks:

```text
kubectl get configmap -n cattle-dashboards dev-connect-gateway-dev01-dashboard
kubectl get configmap -n cattle-dashboards dev-connect-gateway-dev01-dashboard -o yaml
```

Expected labels:

```yaml
grafana_dashboard: "1"
```

Action:

- confirm Rancher Monitoring is installed,
- confirm the `cattle-dashboards` namespace exists,
- wait for the dashboard sidecar sync interval,
- verify chart values for `monitoring.dashboard.enabled`.

### Dashboard Target Filter Empty

Symptoms:

- dashboard opens, but `Server` filter has no values.

Checks:

```text
kubectl get servicemonitor -n dev-connect -o yaml
```

Expected ServiceMonitor endpoint contains:

```yaml
metricRelabelings:
  - targetLabel: dev_connect_target
    replacement: dev01
```

Action:

- wait for Prometheus to scrape the updated ServiceMonitor,
- confirm HAProxy metrics endpoint is reachable on port `8404`,
- confirm the ServiceMonitor selector matches the gateway Service labels.

## Emergency Disable

To stop new sessions for one target:

```text
kubectl scale deployment/dev-connect-gateway-dev01 -n dev-connect --replicas=0
```

Existing client tunnels may fail once the gateway Pod is removed.

To restore:

```text
kubectl scale deployment/dev-connect-gateway-dev01 -n dev-connect --replicas=2
kubectl rollout status deployment/dev-connect-gateway-dev01 -n dev-connect
```

For Helm-managed environments, prefer applying the desired replica count through
Helm or GitOps after emergency mitigation.

## Change Management

All production changes should be traceable through:

- Git commit,
- pull request approval,
- release version,
- Helm release revision,
- deployment timestamp,
- operator identity,
- affected target server.

Changes requiring heightened review:

- RBAC changes,
- NetworkPolicy egress expansion,
- host key rotation,
- gateway target changes,
- monitoring alert threshold changes,
- production replica or resource changes.

## Post-Change Validation

After every production change:

```text
kubectl rollout status deployment/dev-connect-gateway-dev01 -n dev-connect
kubectl get pods -n dev-connect
kubectl get servicemonitor,prometheusrule -n dev-connect
dev-connect --config /path/to/config.yaml config validate
dev-connect --config /path/to/config.yaml connect dev01 --no-code
dev-connect disconnect
```

Record the validation result in the change ticket or pull request.
