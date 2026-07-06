# dev-connect Repository Structure

Status: Approved

## 1. Purpose

This document defines the professional repository structure for `dev-connect`.

Phase 8 creates the repository layout and ownership map. It does not implement Go code, tests, Kubernetes manifests, Helm templates, CI/CD workflows, container images, or release automation.

## 2. Top-Level Structure

```text
dev-connect/
  architecture/
  specification/
  non-functional-requirements/
  kubernetes-design/
  client/
  security/
  testing/
  docs/
  design/
  gateway/
  kubernetes/
  charts/
  tests/
  examples/
  scripts/
  .github/
```

## 3. Existing Approved Specification Areas

`architecture/`

- Phase 1 architecture and review.

`specification/`

- Phase 2 functional specification and review.

`non-functional-requirements/`

- Phase 3 NFRs and review.

`kubernetes-design/`

- Phase 4 Kubernetes design and review.

`client/`

- Phase 5 client design and review.

`security/`

- Phase 6 security specification and review.

`testing/`

- Phase 7 testing strategy and review.

## 4. Implementation-Oriented Areas for Later Phases

`docs/`

- Architecture guide.
- Developer guide.
- Operator guide.
- Installation guide.
- Configuration guide.
- Troubleshooting guide.
- Security guide.
- API documentation.

`design/`

- Cross-cutting design decisions and ADRs.
- Future operator and dynamic gateway design notes.

`gateway/`

- Future gateway container source and HAProxy assets.

`kubernetes/`

- Future Kustomize base and overlays.

`charts/`

- Future Helm chart implementation.

`tests/`

- Future unit, integration, E2E, security, performance, and testdata assets.

`examples/`

- Example client configuration and GitOps inventory examples.

`scripts/`

- Future developer, validation, release, and local test helper scripts.

`.github/`

- Future GitHub Actions workflows and repository automation.

## 5. CI/CD and Release Repository Decisions

### 5.1 Workflow Location

During development inside this repository, CI/CD workflow definitions may reside under:

```text
development/dev-connect/.github/workflows
```

The project architecture assumes that `dev-connect` will eventually become a standalone repository. After extraction, GitHub Actions workflows shall live at the standalone repository root:

```text
.github/workflows
```

Workflow definitions shall not contain assumptions about their current directory structure.

### 5.2 Generated Artifacts

Generated artifacts shall never be committed to Git.

Generated artifacts shall always be rebuilt during CI/CD.

Examples:

- generated Kubernetes manifests,
- generated documentation,
- Helm packages,
- release archives,
- SBOMs,
- binaries,
- test reports,
- coverage reports.

Exceptions may only be made for generated API documentation or examples if explicitly approved.

### 5.3 Branch and Release Strategy

The project shall use a simplified GitHub Flow with semantic versioning.

Branches:

```text
main
feature/<name>
bugfix/<name>
hotfix/<name>
release/<major>.<minor>
```

Branch meaning:

- `main`: production-ready source.
- `feature/<name>`: feature development.
- `bugfix/<name>`: bug fixes.
- `hotfix/<name>`: production fixes.
- `release/<major>.<minor>`: optional stabilization branches for major releases.

Versioning:

```text
v1.0.0
v1.1.0
v1.1.1
```

Pre-releases:

```text
v1.0.0-alpha.1
v1.0.0-beta.1
v1.0.0-rc.1
```

Container images:

```text
ghcr.io/anwendt/dev-connect:v1.0.0
ghcr.io/anwendt/dev-connect:latest
```

Helm chart versions shall match the application version, for example:

```text
1.0.0
```

### 5.4 Repository Principles

The repository shall remain fully reproducible.

Every release shall be generated exclusively from source code using the CI/CD pipeline.

No manually created build artifacts shall be stored in version control.

The repository structure shall allow extraction into an independent open-source project without requiring changes to implementation or pipeline configuration.

## 6. Repository Rules

- Do not place credentials, SSH private keys, Kubernetes tokens, kubeconfigs with credentials, or passwords anywhere in the repository.
- Use `tests/testdata/` only for synthetic fixtures.
- Keep generated artifacts out of source control unless explicitly approved as generated API documentation or examples.
- Keep implementation files out of this repository structure phase.
- Follow Specification Driven Development and Test Driven Development before adding implementation code.

## 7. Phase Gate

This document completes the approved Phase 8 repository structure. Phase 9 may start.
