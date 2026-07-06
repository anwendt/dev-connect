# dev-connect Developer Guide

Status: Draft for Phase 10 review

## Purpose

This guide describes how a developer uses `dev-connect` to open a VS Code Desktop Remote SSH session to a private Linux development server.

## Prerequisites

The developer workstation needs:

- `dev-connect` installed.
- `kubectl` installed and available in `PATH`.
- A working kubeconfig for the target platform.
- Access through the existing Rancher authentication model.
- VS Code Desktop installed.
- OpenSSH client support.
- SSH access to the target Linux development server.

No VPN, bastion host, public IP, browser-based VS Code, or direct Kubernetes API integration in `dev-connect` is required.

## Connect

Example:

```text
dev-connect connect dev01
```

Expected behavior:

1. Load local YAML configuration.
2. Resolve selected context, cluster, gateway, and target.
3. Locate `kubectl`.
4. Verify Kubernetes connectivity through `kubectl`.
5. Verify Kubernetes permissions.
6. Validate that `kubectl port-forward` works.
7. Allocate a free local loopback TCP port.
8. Start managed `kubectl port-forward`.
9. Write temporary SSH config.
10. Write temporary pinned `known_hosts`.
11. Launch VS Code Desktop Remote SSH.
12. Monitor the tunnel and reconnect automatically by default.

## Disconnect

```text
dev-connect disconnect
```

Expected behavior:

- stop the managed `kubectl port-forward`,
- remove temporary SSH config,
- remove temporary `known_hosts`,
- update local JSON session state,
- avoid killing unrelated user processes.

## Status

```text
dev-connect status
```

The status command reports local session metadata such as:

- session ID,
- target,
- gateway,
- Kubernetes context,
- local port,
- tunnel process state,
- reconnect state.

Machine-readable output is available with:

```text
dev-connect status --output json
```

Every JSON response contains an `apiVersion` field.

## List Targets

```text
dev-connect list
```

The list command shows configured targets that the current configuration can resolve.

## VS Code Behavior

`dev-connect` launches VS Code Desktop with Remote SSH. VS Code installs or updates VS Code Server on the target development server according to normal VS Code Remote SSH behavior.

GitHub Copilot continues to run locally inside VS Code Desktop. The gateway does not execute Copilot and does not inspect VS Code or SSH payloads.

## Reconnect

Automatic reconnect is enabled by default.

Reconnect only manages the `kubectl port-forward` tunnel. It does not retry SSH passwords, alter SSH authentication, or bypass target server controls.

To disable reconnect for a session:

```text
dev-connect connect dev01 --no-reconnect
```

## Security Expectations

Developers must not disable SSH host key checking.

The following behavior is forbidden:

```text
StrictHostKeyChecking=no
```

If a host key mismatch occurs, stop and contact the platform team.

