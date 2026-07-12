# dev-connect Phase 8 Review

Status: Approved

## Scope

Phase 8 covers repository structure only. It intentionally does not include Go implementation files, Go tests, Kubernetes manifests, Helm templates, Kustomize resources, CI/CD workflows, binaries, or container images.

Primary artifacts:

- `docs/design/repository-structure.md`
- consolidated top-level repository directories
- documentation directories below `docs/`

## Review Checklist

Structure:

- [x] Approved phase documentation directories are preserved below `docs/`.
- [x] `docs/` contains documentation, design, review, specification, and runbook material.
- [x] `docs/design/` exists for ADRs and design notes.
- [x] `gateway/` exists for future gateway implementation.
- [x] `deploy/kubernetes/` exists for Kustomize resources.
- [x] `charts/` exists for future Helm charts.
- [x] `tests/` exists for future test suites.
- [x] `examples/` exists for future sample configuration.
- [x] `scripts/` exists for future helper scripts.
- [x] `.github/` exists for future GitHub automation.

Safety:

- [x] No credentials are introduced.
- [x] No implementation code is introduced.
- [x] No test code is introduced.
- [x] No Kubernetes manifests are introduced.
- [x] No Helm templates are introduced.
- [x] No GitHub Actions workflows are introduced.

Traceability:

- [x] Repository structure maps to approved phases.
- [x] Documentation indexes explain intended ownership.
- [x] Phase 9 CI/CD work has a reserved location.

## Required Decisions Before Phase 9

The following decisions are approved for Phase 9 CI/CD design:

1. CI/CD workflow location:
   - during development, workflows may reside under `development/dev-connect/.github/workflows`.
   - `dev-connect` is expected to become a standalone repository.
   - after extraction, workflows shall reside at repository root under `.github/workflows`.
   - workflow definitions shall not assume the current directory structure.
2. Generated artifacts:
   - generated artifacts shall never be committed to Git.
   - artifacts shall always be rebuilt during CI/CD.
   - examples include generated Kubernetes manifests, generated documentation, Helm packages, release archives, SBOMs, binaries, test reports, and coverage reports.
   - exceptions are allowed only for generated API documentation or examples when explicitly approved.
3. Branch and release strategy:
   - use simplified GitHub Flow.
   - production-ready source lives on `main`.
   - feature branches use `feature/<name>`.
   - bugfix branches use `bugfix/<name>`.
   - hotfix branches use `hotfix/<name>`.
   - optional stabilization branches use `release/<major>.<minor>`.
   - use Semantic Versioning.
   - release tags use `v<major>.<minor>.<patch>`.
   - pre-releases use `v1.0.0-alpha.1`, `v1.0.0-beta.1`, and `v1.0.0-rc.1` style.
   - container images use `ghcr.io/anwendt/dev-connect:<version>` and `ghcr.io/anwendt/dev-connect:latest`.
   - Helm chart versions match the application version.
4. Repository principles:
   - every release shall be generated exclusively from source code using CI/CD.
   - no manually created build artifacts shall be stored in version control.
   - repository structure shall support future extraction into a standalone open-source project without implementation or pipeline rewrites.

Remaining decisions before Phase 9:

No Phase 8 repository structure decisions remain open.

## Approval Gate

Phase 8 is approved. Phase 9 may start.
