<!--
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2026 Andres Wendt <andres.wendt@wendt.cc>
-->

# dev-connect Troubleshooting Guide

Status: Approved

## kubectl Not Found

Symptoms:

- `connect` fails before Kubernetes checks.

Checks:

- verify a configured `--kubectl-path` or `clusters.<name>.kubectlPath`,
- verify `DEV_CONNECT_KUBECTL_PATH` if it is used,
- verify `kubectl` is next to `dev-connect` when using the Windows bundle,
- verify `kubectl` is in `PATH` or in an OS-specific default location.

Discovery order:

1. `--kubectl-path <path>`
2. `DEV_CONNECT_KUBECTL_PATH`
3. `clusters.<name>.kubectlPath`
4. `kubectl` next to the `dev-connect` executable
5. `kubectl` in `PATH`
6. OS-specific default locations

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

- SSH keepalive is enabled by default in generated SSH configuration.
- A local `dev-connect tunnel-supervisor` process is started by `connect`.
- The supervisor restarts `kubectl port-forward` after transient process exits
  unless `--no-reconnect` was used.

Checks:

- confirm Kubernetes API health,
- check local network stability,
- verify gateway Pod restarts or node disruption events,
- lower `ssh.serverAliveInterval` when a proxy closes idle connections,
- verify the generated SSH config contains `ServerAliveInterval`,
  `ServerAliveCountMax`, and `TCPKeepAlive`,
- verify `dev-connect status --output json` reports an active session PID,
- inspect the local `supervisor.log` in the dev-connect session directory.

Example for aggressive proxy idle timeouts:

```yaml
ssh:
  serverAliveInterval: 15
  serverAliveCountMax: 4
  tcpKeepAlive: true
```

## Cleanup Problems

Expected cleanup:

- managed `kubectl port-forward` process stops,
- temporary SSH config is removed,
- temporary `known_hosts` is removed,
- JSON session state is updated.

If stale state remains, `disconnect` and `status` should report enough metadata to identify stale resources without deleting unrelated processes.
