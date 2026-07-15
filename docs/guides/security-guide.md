# dev-connect Security Guide

Status: Approved

## Security Model

`dev-connect` separates Kubernetes transport authorization from SSH authentication.

Kubernetes controls who may open a port-forward to the gateway. The target Linux server controls who may authenticate over SSH.

## Mandatory Controls

- No public endpoints.
- No VPN or bastion host.
- No SSH authentication in the gateway.
- No SSH credentials in Kubernetes.
- No Kubernetes tokens written by `dev-connect`.
- No direct Rancher or Kubernetes API client in `dev-connect`.
- SSH host key pinning is mandatory.
- Logs are metadata-only.
- Developer RBAC is least privilege.

## Identity and Access

Kubernetes authentication and authorization follow the existing Rancher authentication model.

The solution does not introduce:

- local Kubernetes users,
- separate identity store,
- user database,
- credential database.

Access is granted through Rancher-managed Kubernetes RBAC and existing enterprise identity providers where configured.

## SSH Security

The gateway is not an SSH proxy in the authentication sense. It is a TCP forwarder.

The gateway shall not:

- terminate SSH,
- inspect passwords or keys,
- validate users,
- store user records,
- log SSH payloads.

The target development server performs SSH authentication through standard OpenSSH.

## Host Key Pinning

The client writes expected host keys into a temporary `known_hosts` file before launching VS Code Remote SSH.

Host keys are managed through the enterprise Platform GitOps repository under a dedicated `dev-connect` path.

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

The concrete repository path is defined by Platform Engineering during production onboarding.

Host key rotation requires normal platform pull request approval by at least two authorized platform administrators.

## Logging and Audit

Allowed metadata:

- user identity where available,
- target,
- session ID,
- gateway,
- namespace,
- Kubernetes context,
- local port,
- start time,
- end time,
- duration,
- exit code.

Forbidden log content:

- passwords,
- SSH private keys,
- bearer tokens,
- Kubernetes tokens,
- proxy credentials,
- kubeconfig credentials,
- SSH payload,
- terminal contents,
- transferred file contents.

## Supply Chain

The CI/CD design requires:

- Go 1.26.x,
- `golangci-lint`,
- `govulncheck`,
- Trivy,
- `go-licenses`,
- Syft SPDX 2.3 SBOMs,
- Cosign signing,
- Renovate dependency updates,
- SLSA Level 3 compatibility target where feasible,
- provenance generation.

## Security Testing

Mandatory CI quality gates include:

- RBAC static validation,
- forbidden Kubernetes/Rancher import checks,
- no direct HTTP(S) communication to Rancher or Kubernetes API,
- metadata log redaction tests,
- host key rotation tests,
- security scan gates.
