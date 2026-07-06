# dev-connect GitHub Automation

This directory contains GitHub-specific project automation for `dev-connect`.

For the main project overview, purpose, architecture, quick start, security
model, release process, and documentation links, see the repository root:

[dev-connect README](../README.md)

## Project Purpose

`dev-connect` enables developers to use Microsoft Visual Studio Code Desktop
with Remote SSH against private-cloud Linux development servers through an
existing Kubernetes cluster.

It is intended for environments where:

- development servers are outside Kubernetes,
- direct workstation access to the private cloud is not allowed,
- no VPN, bastion host, public IP address, or browser-based IDE is permitted,
- all transport to the private cloud must go through the Kubernetes API,
- SSH authentication remains on the target Linux development server.

## Automation Contents

```text
.github/
  workflows/
    ci.yml
    release.yml
    security-scan.yml
    performance.yml
```

## Workflow Summary

| Workflow | Purpose |
| --- | --- |
| `ci.yml` | Validates client builds, tests, linting, and Kubernetes packaging on main and pull requests. |
| `release.yml` | Builds release binaries, packages and publishes the Helm chart, creates SBOMs, signs artifacts, and publishes release outputs. |
| `security-scan.yml` | Runs security and architecture compliance checks. |
| `performance.yml` | Provides manually triggered performance validation. |

The release workflow publishes:

- Windows, Linux, and macOS client binaries,
- Helm chart package,
- Helm OCI chart in GHCR,
- SBOM,
- checksums,
- provenance attestations,
- GitHub Release artifacts.

All workflow commands are routed through Makefile targets to keep local and CI
execution consistent.
