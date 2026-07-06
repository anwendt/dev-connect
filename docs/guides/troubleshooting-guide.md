# dev-connect Troubleshooting Guide

Status: Draft for Phase 10 review

## kubectl Not Found

Symptoms:

- `connect` fails before Kubernetes checks.

Checks:

- verify `kubectl` is installed,
- verify `kubectl` is in `PATH`,
- verify configured `kubectl` path if an override is used.

## Kubernetes Authentication Fails

Symptoms:

- `kubectl` reports authentication errors,
- Rancher login or token flow fails.

Checks:

- run `kubectl config current-context`,
- verify the selected kubeconfig,
- refresh Rancher or enterprise identity login,
- confirm enterprise proxy configuration.

## Kubernetes Authorization Fails

Symptoms:

- `kubectl auth can-i` fails,
- temporary port-forward validation fails.

Likely causes:

- missing Rancher-managed RBAC group membership,
- missing namespace access,
- missing `pods/portforward create`,
- admission controller restriction.

Action:

- contact platform operators with the target, namespace, Kubernetes context, and session ID if available.

## Port Forward Does Not Establish

Symptoms:

- local port is allocated but tunnel never becomes ready,
- `kubectl port-forward` exits.

Checks:

- gateway Service exists,
- gateway Pods are ready,
- Service endpoints exist,
- local port is not blocked,
- Kubernetes API server is healthy,
- platform proxy allows port-forward streaming.

## VS Code Does Not Launch

Checks:

- verify `code` exists in `PATH`,
- verify OS-specific default install path,
- configure explicit VS Code launcher path,
- run with `--no-code` to validate tunnel setup independently.

## SSH Host Key Mismatch

Do not bypass host key checking.

Forbidden workaround:

```text
StrictHostKeyChecking=no
```

Action:

- stop the connection attempt,
- compare with GitOps-managed host key inventory,
- contact platform operators,
- rotate or correct the host key through the approved change process.

## Tunnel Drops

Expected behavior:

- automatic reconnect starts by default for managed tunnel failures.

Checks:

- confirm Kubernetes API health,
- check local network stability,
- inspect metadata logs for reconnect count and exit code,
- verify gateway Pod restarts or node disruption events.

## Cleanup Problems

Expected cleanup:

- managed `kubectl port-forward` process stops,
- temporary SSH config is removed,
- temporary `known_hosts` is removed,
- JSON session state is updated.

If stale state remains, `disconnect` and `status` should report enough metadata to identify stale resources without deleting unrelated processes.

