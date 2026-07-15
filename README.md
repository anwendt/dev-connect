# dev-connect

`dev-connect` is an enterprise development gateway for using Microsoft Visual
Studio Code Desktop with Remote SSH against Linux development servers located in
a private cloud.

It is designed for environments where developer workstations are not allowed to
connect directly to private cloud networks, and where the only approved
communication path is through an existing Kubernetes cluster.

## Purpose

`dev-connect` lets developers keep their local desktop development experience:

- VS Code Desktop
- GitHub Copilot running locally
- locally installed VS Code extensions
- standard OpenSSH authentication
- enterprise workstation proxy settings

while opening Remote SSH sessions to private Linux development servers that are
not running inside Kubernetes.

No VPN, bastion host, public IP address, browser-based IDE, or SSH credentials
inside Kubernetes are required.

## What It Does

A developer runs:

```text
dev-connect connect dev01
```

The client then:

1. Loads the local `config.yaml` configuration.
2. Resolves the target development server and Kubernetes gateway.
3. Verifies `kubectl` connectivity.
4. Verifies Rancher/Kubernetes RBAC permissions.
5. Starts `kubectl port-forward` to the gateway Service.
6. Generates temporary SSH configuration and `known_hosts` files.
7. Enforces pinned SSH host key verification.
8. Launches VS Code Desktop Remote SSH.
9. Monitors the tunnel and supports automatic reconnect.
10. Cleans up temporary local resources on disconnect.

All Kubernetes communication is delegated to the locally installed `kubectl`
binary using the user's existing kubeconfig and enterprise authentication model.
The Go client does not directly call the Kubernetes API or Rancher API.

## Architecture

The first implementation uses:

- a cross-platform Go CLI client,
- `kubectl port-forward` for Kubernetes API transport,
- a Kubernetes-hosted HAProxy TCP gateway,
- one gateway deployment per target development server,
- ClusterIP-only Kubernetes Services,
- NetworkPolicy-based egress restrictions,
- Rancher-managed Kubernetes RBAC,
- pinned SSH host keys managed outside Kubernetes,
- Rancher Monitoring compatible ServiceMonitor, PrometheusRule, and Grafana dashboard resources.

The gateway only forwards TCP traffic. It does not terminate SSH, authenticate
SSH users, inspect payloads, store credentials, or maintain a user database.

SSH authentication always happens on the target Linux development server.

## Repository Layout

```text
charts/dev-connect-gateway/    Helm chart for the HAProxy gateway
cmd/dev-connect/               Go CLI entry point
deploy/kubernetes/             Kustomize resources
docs/                          User, operator, security, API, and runbook docs
examples/config/               Validated client configuration examples
gateway/haproxy/               Gateway container assets
internal/                      Go implementation packages
scripts/                       Developer and release helper scripts
tests/                         Integration, security, E2E, and performance tests
```

## Quick Start

### Build the Client

```text
make build
```

Cross-platform release binaries:

```text
make build-all VERSION=dev
```

### Validate the Project

```text
make test
make lint
make helm-lint
make helm-template
make kustomize-build
```

### Validate a Client Configuration

```text
dev-connect --config examples/config/dev01-basic.yaml config validate
```

### Connect Without Launching VS Code

```text
dev-connect --config examples/config/dev01-basic.yaml connect dev01 --no-code
dev-connect status
dev-connect disconnect
```

## Client Configuration

Validated examples are available in:

```text
examples/config/
```

The examples include:

- a minimal `dev01` setup,
- process-scoped proxy override settings for `kubectl`,
- a multi-cluster configuration,
- a Windows setup with bundled `kubectl.exe`.

Default configuration locations:

| Operating system | Default path |
| --- | --- |
| Windows | `%APPDATA%\dev-connect\config.yaml` |
| Linux | `~/.config/dev-connect/config.yaml` |
| macOS | `~/Library/Application Support/dev-connect/config.yaml` |

The effective configuration location can be shown with:

```text
dev-connect config location
```

`dev-connect` delegates Kubernetes access to `kubectl`. The client resolves
`kubectl` from `--kubectl-path`, `DEV_CONNECT_KUBECTL_PATH`,
`clusters.<name>.kubectlPath`, a `kubectl` binary next to `dev-connect`, `PATH`,
and finally documented default locations. The Windows bundle release artifact
contains both `dev-connect.exe` and `kubectl.exe`, so no global `kubectl`
installation is required on Windows clients.

## Kubernetes Gateway

The gateway is deployed with Helm or Kustomize.

Example Helm installation:

```text
helm upgrade --install dev-connect-dev01 charts/dev-connect-gateway \
  --namespace dev-connect \
  --create-namespace \
  --set target.name=dev01 \
  --set target.host=172.28.192.14 \
  --set target.port=22 \
  --set networkPolicy.backendCIDR=172.28.192.14/32 \
  --set networkPolicy.dnsEgress.enabled=false \
  --set monitoring.enabled=true
```

Release builds also publish the Helm chart as an OCI artifact:

```text
oci://ghcr.io/anwendt/charts/dev-connect-gateway
```

## Security Model

`dev-connect` follows a least-privilege model:

- no public Kubernetes endpoint is created,
- no VPN or bastion host is required,
- no SSH credentials are stored in Kubernetes,
- no SSH keys are stored in Kubernetes,
- no user database exists in Kubernetes,
- the gateway performs TCP forwarding only,
- SSH authentication remains on the target Linux server,
- host key verification is mandatory,
- `StrictHostKeyChecking=no` is forbidden,
- logs contain metadata only and never SSH payloads.

Developer Kubernetes access is granted through Rancher-managed Kubernetes RBAC
and the existing enterprise identity provider configuration.

## Observability

The Helm chart can install Rancher Monitoring compatible resources:

- `ServiceMonitor`
- `PrometheusRule`
- Grafana dashboard ConfigMap

The dashboard is named:

```text
dev-connect Gateway
```

The target/server filter is:

```text
Server
```

The underlying Prometheus label is:

```text
dev_connect_target
```

## Release Pipeline

The GitHub Actions release workflow builds and publishes:

- Windows, Linux, and macOS client binaries,
- Windows client bundle with `dev-connect.exe` and `kubectl.exe`,
- Helm chart package,
- Helm OCI chart in GHCR,
- container image in GHCR,
- SBOM,
- checksums,
- container image build and scan,
- provenance attestations,
- GitHub Release artifacts.

Release artifacts are generated from source during CI/CD. Generated artifacts
are not committed to Git.

## Documentation

Important documentation entry points:

- [Architecture Guide](docs/guides/architecture-guide.md)
- [Developer Guide](docs/guides/developer-guide.md)
- [Operator Guide](docs/guides/operator-guide.md)
- [Development Server Onboarding Guide](docs/guides/development-server-onboarding-guide.md)
- [Configuration Guide](docs/guides/configuration-guide.md)
- [Security Guide](docs/guides/security-guide.md)
- [Troubleshooting Guide](docs/guides/troubleshooting-guide.md)
- [Operations Runbook](docs/runbooks/operations-runbook.md)
- [API Documentation](docs/api/api-documentation.md)

## Project Status

The specification, architecture, security model, testing strategy, repository
structure, CI/CD design, documentation, Go client foundation, Kubernetes
gateway, Helm chart, Kustomize resources, monitoring integration, and automated
tests are implemented.

The current automated Go test coverage is above 80% statements.

Production rollout still requires environment-specific onboarding:

- approved production host key inventory,
- final Rancher RBAC group bindings,
- production Helm values,
- enterprise logging and dashboard ownership,
- production-like end-to-end validation,
- formal production load testing.
