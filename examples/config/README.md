# dev-connect Config Examples

This directory contains validated example configuration files for `dev-connect`.

All examples use the supported configuration API:

```yaml
apiVersion: dev-connect/v1
kind: DevConnectConfig
```

The examples are intentionally usable as starting points, but pinned SSH host
keys are placeholders. Replace every `hostKeys` value with the approved host key
from the enterprise GitOps host key inventory before using a configuration.

## Examples

| File | Purpose |
| --- | --- |
| `dev01-basic.yaml` | Minimal single-cluster configuration without proxy override. |
| `dev01-proxy.yaml` | Single-cluster configuration with a process-scoped proxy override for `kubectl`. |
| `multi-cluster-proxy.yaml` | Multiple Rancher/Kubernetes contexts, multiple gateways, and per-cluster proxy settings. |

## Usage

Validate an example:

```text
dev-connect --config examples/config/dev01-basic.yaml config validate
```

Connect without launching VS Code:

```text
dev-connect --config examples/config/dev01-basic.yaml connect dev01 --no-code
```

Connect with machine-readable output:

```text
dev-connect --config examples/config/dev01-proxy.yaml --output json connect dev01 --no-code
```

Select a named dev-connect context:

```text
dev-connect --config examples/config/multi-cluster-proxy.yaml --context central-dev connect dev01
```

## Proxy Behavior

Proxy settings under `clusters.<name>.proxy` are optional.

When `enabled: false`, `kubectl` inherits the normal user environment, including
the enterprise proxy configuration already configured on the workstation.

When `enabled: true`, `dev-connect` applies the configured values only to the
`kubectl` process it starts. It does not modify:

- operating system proxy settings,
- shell profile files,
- kubeconfig,
- corporate proxy configuration,
- global environment variables.

Use a proxy override only when the default enterprise workstation proxy settings
are not sufficient for the selected Kubernetes or Rancher endpoint.
