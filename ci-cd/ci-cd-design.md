# dev-connect CI/CD Design

Status: Approved

## 1. Purpose

This document defines the CI/CD design for `dev-connect`.

Phase 9 describes GitHub Actions workflows and release automation at design level only. It does not create workflow YAML files, Go code, tests, Kubernetes manifests, Helm templates, container images, binaries, SBOMs, signatures, or release artifacts.

## 2. CI/CD Principles

- CI/CD shall be reproducible from source.
- Generated artifacts shall not be committed to Git.
- Generated artifacts shall be rebuilt in CI/CD.
- CI/CD workflow definitions shall not assume the current mono-repository path.
- The project shall be extractable into a standalone repository without pipeline rewrites.
- Security and architecture compliance tests are mandatory quality gates.
- Releases shall use Semantic Versioning.
- Container images and Helm charts shall be produced by CI/CD only.
- Local developer workflows and CI/CD workflows shall use the same Makefile targets.
- GitHub Actions shall call Makefile targets instead of duplicating build, test, scan, package, and release command logic.
- Every release shall be built from source.
- No manually created artifacts shall be published.
- All security scans shall execute before release publication.
- Container images, SBOMs, and signatures shall be generated from the same binary artifact.
- Dependency updates shall be automated where practical.
- Release pipelines shall target SLSA Level 3 compatibility where feasible.

## 3. Workflow Location

During development, workflow definitions may reside under:

```text
development/dev-connect/.github/workflows
```

After extraction to a standalone repository, workflows shall reside under:

```text
.github/workflows
```

Workflow implementation shall avoid hard-coded assumptions about the current path.

## 4. Pipeline Overview

Required stages:

1. Lint.
2. Formatting.
3. Unit tests.
4. Integration tests.
5. Security scan.
6. Container build.
7. Helm lint.
8. Helm package.
9. Release.
10. SBOM.
11. Container signing.

Recommended pipeline groups:

- Pull request validation.
- Main branch validation.
- Release build.
- Manual performance validation.
- Scheduled security scan.

## 4.1 Makefile-Based Execution Model

The `Makefile` shall be the canonical command interface for local development and CI/CD automation.

GitHub Actions workflow steps shall invoke `make` targets. Workflow YAML shall orchestrate stages, secrets, permissions, caching, matrix execution, and artifact upload, but shall not duplicate implementation command logic that belongs in Makefile targets.

Required Makefile target principles:

- Targets shall be deterministic and safe to execute locally.
- Targets shall be documented through a `help` target.
- Targets used by pull request validation shall not require secrets.
- Targets requiring credentials, publishing, signing, or release permissions shall be clearly separated from validation targets.
- Targets shall avoid assumptions about the current mono-repository path so the project can be extracted into a standalone repository.
- Targets shall return non-zero exit codes on validation failures.

Baseline Makefile targets:

- `make help`: list supported targets.
- `make fmt`: format source files.
- `make fmt-check`: verify formatting without modifying files.
- `make lint`: run linters.
- `make test-unit`: run unit tests.
- `make test-integration`: run integration tests.
- `make test-security`: run security and architecture compliance tests.
- `make test`: run the default validation test suite.
- `make test-performance-local`: run local benchmarks and lightweight performance checks.
- `make build`: build local binaries.
- `make container-build`: build the gateway container image.
- `make helm-lint`: run Helm lint checks.
- `make helm-template`: render Helm templates for validation.
- `make helm-package`: package the Helm chart.
- `make sbom`: generate SBOM artifacts.
- `make sign`: sign release artifacts and container images.
- `make release-check`: validate release readiness.
- `make clean`: remove local generated build outputs.

CI/CD stage mapping:

| CI/CD stage | Makefile target |
|-------------|-----------------|
| Formatting | `make fmt-check` |
| Lint | `make lint` |
| Unit tests | `make test-unit` |
| Integration tests | `make test-integration` |
| Security scan | `make test-security` |
| Container build | `make container-build` |
| Helm lint | `make helm-lint` |
| Helm template validation | `make helm-template` |
| Helm package | `make helm-package` |
| SBOM | `make sbom` |
| Container and artifact signing | `make sign` |
| Release readiness | `make release-check` |

## 5. Pull Request Validation

Purpose:

- Fast feedback.
- Prevent regressions before merge.
- Enforce mandatory quality gates.

Required checks:

- Go formatting check.
- Go lint.
- Go unit tests.
- Go integration tests using fake `kubectl`, mock SSH server, and mock VS Code launcher.
- Architecture compliance checks:
  - no direct Kubernetes client imports,
  - no Rancher client imports,
  - Kubernetes operations go through `kubectl`.
- Security tests:
  - RBAC static validation,
  - log redaction tests,
  - host key validation tests.
- Dependency license check.
- Generated artifact cleanliness check.

PR checks shall fail on:

- test failures,
- formatting failures,
- lint failures,
- forbidden imports,
- sensitive log output,
- RBAC privilege creep,
- committed generated artifacts,
- forbidden licenses.

## 6. Main Branch Validation

Purpose:

- Validate merged source.
- Produce non-release build artifacts where appropriate.
- Maintain confidence in `main`.

Required checks:

- All pull request checks.
- Broader integration test suite.
- Container build dry run or non-release image build.
- Helm template rendering.
- Helm lint.
- SBOM generation dry run.
- Vulnerability scan.

Artifacts:

- CI artifacts may be uploaded to GitHub Actions retention storage.
- Artifacts shall not be committed to Git.

## 7. Release Pipeline

Trigger:

- Semantic version tag such as `v1.0.0`.
- Pre-release tags such as `v1.0.0-alpha.1`, `v1.0.0-beta.1`, `v1.0.0-rc.1`.

Release outputs:

- Cross-platform `dev-connect` binaries.
- Container image.
- Helm chart package.
- Helm chart OCI artifact in GHCR.
- SBOM.
- Signatures.
- GitHub Release notes.

Release build order:

1. Compile.
2. Unit tests.
3. Integration tests.
4. Binary build.
5. SBOM generation.
6. Container build.
7. Container scan.
8. Helm chart package.
9. Helm OCI publication.
10. Signing.
11. Release publication.

Release binaries shall be built first. Container images shall consume the released binary artifacts instead of compiling the application again inside the image build.

Versioning:

- Application version follows SemVer.
- Release tags use `v<major>.<minor>.<patch>`.
- Helm chart version matches application version without leading `v`.
- Container images use:

```text
ghcr.io/anwendt/dev-connect:v1.0.0
ghcr.io/anwendt/dev-connect:latest
```

The default registry is:

```text
ghcr.io/anwendt/dev-connect
```

The GHCR organization is `anwendt` for the initial implementation.

Helm charts shall also be published as OCI artifacts:

```text
oci://ghcr.io/anwendt/charts/dev-connect-gateway
```

`latest` shall only be updated for stable releases, not alpha, beta, or release candidate builds.

## 8. Build Matrix

Client binary targets:

- Windows amd64.
- Windows arm64 where supported.
- Linux amd64.
- Linux arm64.
- macOS amd64.
- macOS arm64.

Gateway container:

- Linux amd64.
- Linux arm64 where practical.

## 9. Go Version

CI shall use the currently supported stable Go release approved for the project.

The initial implementation shall target:

```text
Go 1.26.x
```

Go does not have a classic LTS model. The project shall track supported Go releases and update regularly. Minor version updates shall be handled through automated dependency updates where practical.

## 10. Linting and Formatting

Required:

- `gofmt` or `gofmt`-equivalent formatting check.
- Go vet.
- `golangci-lint`.
- Markdown linting should be considered for documentation phases.
- Helm lint for chart phases.

The `golangci-lint` configuration shall be maintained in:

```text
.golangci.yml
```

Recommended linters:

- `govet`.
- `staticcheck`.
- `errcheck`.
- `ineffassign`.
- `gofmt`.
- `goimports`.
- `revive`.
- `misspell`.

Formatting failures shall fail CI.

## 11. Testing in CI

Unit tests:

- Go standard library `testing`.
- `testify` for assertions.
- No Ginkgo for initial implementation.

Integration tests:

- Fake `kubectl` Go helper binary.
- Mock SSH server.
- Mock VS Code launcher.
- No real `~/.kube/config`.
- No real `kubectl`.
- No real VS Code.
- No internet dependency.

Security tests:

- RBAC static validation.
- Forbidden import checks.
- Log redaction tests.
- Host key rotation tests.

Performance tests:

- Local benchmark tests use `go test -bench`.
- Manual CI performance workflow is triggered manually.
- Dedicated performance environment is used for 50-100 session load-test smoke tests.
- Performance tests shall not run automatically on every PR.

## 12. Security Scanning

Required security checks:

- Dependency vulnerability scanning.
- Container image scanning.
- Secret scanning where platform tooling is available.
- License scanning.
- Static analysis for forbidden Kubernetes/Rancher client imports.
- Static RBAC validation.

Security checks shall be mandatory for pull requests and releases where practical.

Approved vulnerability scanning tools:

- `govulncheck` for Go dependencies.
- Trivy for container images and OS packages.

Approved license scanning tool:

- `go-licenses`.

CI shall fail if dependencies use licenses outside the approved policy.

Approved dependency licenses:

- Apache 2.0.
- MIT.
- BSD-2-Clause.
- BSD-3-Clause.
- ISC.

## 13. Container Build

Gateway container requirements:

- Minimal base image.
- Distroless where practical.
- Non-root runtime.
- No `NET_BIND_SERVICE` required.
- HAProxy listens on container port `2222`.
- Service exposes port `22`.

Build requirements:

- Multi-arch support where practical.
- Reproducible build inputs.
- Image labels for source, version, revision, license, and creation time.
- Image pushed to GHCR only from trusted release workflows.
- Release container images shall consume CI-built release binaries.

## 14. Helm Lint and Package

Helm checks:

- `helm lint`.
- `helm template` rendering.
- Values schema validation where implemented.
- Security assertions:
  - no `LoadBalancer`,
  - no `NodePort`,
  - no Ingress,
  - no SSH credentials,
  - `automountServiceAccountToken: false`,
  - HPA disabled by default,
  - Service port `22` target port `2222`.

Packaging:

- Helm chart packages are generated in CI/CD.
- Helm chart packages are not committed.
- Chart version matches application version.

## 15. SBOM

SBOM requirements:

- Generate SBOM for release binaries where practical.
- Generate SBOM for container image.
- Attach SBOM to release artifacts.
- Do not commit generated SBOMs.
- Generate SPDX 2.3 SBOMs.

Approved SBOM tool:

```text
Syft
```

Generated SBOM artifact name:

```text
sbom.spdx.json
```

## 16. Signing

Signing requirements:

- Sign container images.
- Sign release artifacts where tooling supports it.
- Use Cosign.
- Use keyless signing with Sigstore for public releases.
- Support key-based signing as optional enterprise configuration.
- Store signatures as release artifacts or registry metadata.
- Do not commit signatures to Git.

## 17. GitHub Actions Permissions

GitHub Actions permissions shall follow the Principle of Least Privilege.

Default workflow permissions:

```yaml
contents: read
```

Additional permissions shall only be granted to workflows that require them.

Examples:

```yaml
# Release publication
contents: write

# Container image publishing
packages: write

# Keyless signing
id-token: write
```

All other permissions shall be set to `none` or omitted according to GitHub Actions best practices.

## 18. Artifact Retention

Default GitHub Actions artifact retention:

```text
30 days
```

Temporary artifacts include:

- test reports,
- coverage reports,
- intermediate binaries,
- logs.

Release artifacts shall be retained permanently through the release and registry systems.

Release artifacts include:

- GitHub Releases,
- Helm charts,
- OCI images,
- SBOMs,
- signatures.

## 19. Dependency Updates

Automated dependency updates shall use Renovate Bot.

Renovate shall cover:

- Go modules.
- GitHub Actions.
- Docker base images.

Renovate configuration shall be maintained in source control during implementation. Dependency update pull requests shall run the same Makefile-based validation targets as normal pull requests.

## 20. Supply Chain Security

The release pipeline shall target SLSA Level 3 compatibility where feasible.

Required supply chain controls:

- GitHub OIDC for trusted identity in release workflows.
- Reproducible build inputs.
- Signed artifacts using Cosign.
- Signed container images using Cosign.
- Provenance generation for release artifacts and container images.
- SBOM generation from the same release build inputs.
- Least-privilege GitHub Actions permissions.

SLSA Level 3 compatibility is a target for the release pipeline. Any gaps that cannot be closed in the initial implementation shall be documented with compensating controls and follow-up work.

## 21. Release Notes

Release notes shall include:

- Version.
- Commit SHA.
- Summary of changes.
- Security notes.
- Upgrade notes.
- Known issues.
- Checksums and artifact list.

Release notes shall be generated from source-controlled metadata and CI context where practical.

## 22. Branch and Release Rules

Branches:

- `main`: production-ready source.
- `feature/<name>`: feature development.
- `bugfix/<name>`: bug fixes.
- `hotfix/<name>`: production fixes.
- `release/<major>.<minor>`: optional stabilization branches.

Tags:

- Stable: `v1.0.0`.
- Alpha: `v1.0.0-alpha.1`.
- Beta: `v1.0.0-beta.1`.
- Release candidate: `v1.0.0-rc.1`.

## 23. Generated Artifact Policy

Generated artifacts shall not be committed.

CI/CD rebuilds:

- binaries,
- container images,
- Helm packages,
- rendered manifests,
- generated documentation,
- SBOMs,
- signatures,
- test reports,
- coverage reports,
- release archives.

Exceptions require explicit approval for generated API documentation or examples.

## 24. Required Decisions Before Workflow Implementation

The following decisions are approved for the initial CI/CD implementation:

1. CI Go version: Go 1.26.x.
2. Lint tool: `golangci-lint` with `.golangci.yml`.
3. Vulnerability scanning: `govulncheck` and Trivy.
4. License scanning: `go-licenses`.
5. SBOM format and tool: SPDX 2.3 generated by Syft as `sbom.spdx.json`.
6. Signing: Cosign with keyless Sigstore signing for public releases and optional enterprise key-based signing.
7. GHCR organization: `ghcr.io/anwendt/dev-connect`.
8. Build order: release binaries are built before container images; container images consume the release binaries.
9. GitHub Actions permissions: least privilege, starting with `contents: read` and adding `contents: write`, `packages: write`, or `id-token: write` only where required.
10. Artifact retention: 30 days for temporary CI artifacts; release artifacts retained permanently through release and registry systems.
11. Dependency updates: Renovate Bot for Go modules, GitHub Actions, and Docker base images.
12. Supply chain security: release pipeline targets SLSA Level 3 compatibility where feasible using GitHub OIDC, reproducible builds, Cosign signatures, and provenance generation.

No Phase 9 CI/CD design decisions remain open before workflow YAML implementation.

## 25. Phase Gate

This document completes the Phase 9 CI/CD design draft. Phase 10 documentation generation may start only after this CI/CD design is reviewed and approved.
