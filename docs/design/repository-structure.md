# dev-connect Repository Structure

Status: Approved, consolidated

## 1. Purpose

This document defines the maintained repository structure for `dev-connect`.

The repository keeps implementation-oriented directories at the root and moves
architecture, specification, review, and design documents below `docs/`. This
keeps the root small while preserving Specification Driven Development and
auditability.

## 2. Top-Level Structure

```text
dev-connect/
  .github/
  charts/
  cmd/
  deploy/
  docs/
  examples/
  gateway/
  internal/
  scripts/
  tests/
  Makefile
  README.md
  go.mod
  go.sum
```

## 3. Root Directory Ownership

`.github/`

- GitHub Actions workflows and repository automation.

`charts/`

- Helm charts and chart source files.

`cmd/`

- Go CLI entry points.

`deploy/`

- Non-Helm deployment resources.
- Kustomize bases and overlays live under `deploy/kubernetes/`.

`docs/`

- User documentation, operator documentation, API documentation, runbooks,
  architecture, specifications, design decisions, reviews, and implementation
  plans.

`examples/`

- User-facing configuration examples and GitOps inventory examples.

`gateway/`

- Gateway container source and HAProxy assets.

`internal/`

- Go implementation packages.

`scripts/`

- Developer, validation, release, and local helper scripts.

`tests/`

- Integration, E2E, security, performance, fixtures, and supporting test
  assets.

## 4. Documentation Structure

Approved phase artifacts and design material are consolidated under `docs/`:

```text
docs/
  architecture/
  specification/
  non-functional-requirements/
  design/
    ci-cd/
    client/
    general/
    kubernetes/
  implementation/
  security/
  testing/
  guides/
  runbooks/
  api/
```

The documentation structure intentionally avoids many root-level phase
directories. New design documents should be added under `docs/design/` unless
they are user-facing guides, runbooks, API documentation, or security
documentation.

## 5. CI/CD and Release Repository Decisions

### 5.1 Workflow Location

During development inside a monorepository, CI/CD workflow definitions may
reside under:

```text
development/dev-connect/.github/workflows
```

The project architecture assumes that `dev-connect` can become a standalone
repository. After extraction, GitHub Actions workflows shall live at the
standalone repository root:

```text
.github/workflows
```

Workflow definitions shall not contain assumptions about their current parent
directory structure.

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

Exceptions may only be made for generated API documentation or examples if
explicitly approved.

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

Helm chart versions shall follow semantic versioning and declare the matching
application version through `appVersion`.

## 6. Repository Rules

- Do not place credentials, SSH private keys, Kubernetes tokens, kubeconfigs
  with credentials, or passwords anywhere in the repository.
- Use `tests/testdata/` only for synthetic fixtures.
- Keep generated artifacts out of source control unless explicitly approved as
  generated API documentation or examples.
- Keep root-level directories limited to implementation, deployment, examples,
  tests, automation, and documentation entry points.
- Follow Specification Driven Development and Test Driven Development before
  adding implementation code.
