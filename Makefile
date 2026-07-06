.PHONY: help fmt fmt-check lint license-check test-unit test-integration test-security test-e2e test-performance-local test build build-all checksums container-build container-scan helm-lint helm-template helm-package kustomize-build sbom sign release-check clean

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_CACHE ?= $(CURDIR)/.cache/golangci-lint
GO_CACHE ?= $(CURDIR)/.cache/go-build
HELM ?= helm
KUSTOMIZE ?= kustomize
DOCKER ?= docker
SYFT ?= syft
COSIGN ?= cosign
GO_LICENSES ?= go-licenses
TRIVY ?= trivy
IMAGE ?= ghcr.io/anwendt/dev-connect:local
VERSION ?= dev

help:
	@printf '%s\n' \
		'dev-connect targets:' \
		'  make fmt              Format Go source files' \
		'  make fmt-check        Verify Go formatting' \
		'  make lint             Run golangci-lint when available' \
		'  make license-check    Run dependency license policy checks when available' \
		'  make test-unit        Run unit tests' \
		'  make test-integration Run integration tests' \
		'  make test-security    Run security and architecture tests' \
		'  make test-e2e         Run end-to-end tests with local fakes' \
		'  make test-performance-local Run local benchmark checks' \
		'  make test             Run default validation tests' \
		'  make container-build  Build gateway container image' \
		'  make helm-lint        Run Helm chart linting' \
		'  make helm-template    Render Helm chart templates' \
		'  make helm-package     Package Helm chart into dist/' \
		'  make kustomize-build  Render Kustomize base and autoscaling overlay' \
		'  make sbom             Generate SPDX SBOM into dist/' \
		'  make sign             Sign container image when cosign is available' \
		'  make release-check    Validate release readiness' \
		'  make build            Build dev-connect' \
		'  make build-all        Build release binaries for Windows, Linux, and macOS' \
		'  make checksums        Generate SHA-256 checksums for dist artifacts' \
		'  make container-scan   Scan the container image when Trivy is available' \
		'  make clean            Remove local build outputs'

fmt:
	GOCACHE=$(GO_CACHE) $(GO) fmt ./...

fmt-check:
	@test -z "$$(GOCACHE=$(GO_CACHE) $(GO) fmt ./...)" || (printf '%s\n' 'Go files need formatting. Run make fmt.' && exit 1)

lint:
	@if command -v $(GOLANGCI_LINT) >/dev/null 2>&1; then \
		GOCACHE=$(GO_CACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) $(GOLANGCI_LINT) run ./...; \
	else \
		printf '%s\n' 'golangci-lint not installed; skipping lint in local foundation slice.'; \
	fi

license-check:
	@if command -v $(GO_LICENSES) >/dev/null 2>&1; then \
		$(GO_LICENSES) check ./... --allowed_licenses=Apache-2.0,MIT,BSD-2-Clause,BSD-3-Clause,ISC; \
	else \
		printf '%s\n' 'go-licenses not installed; skipping license-check in local slice.'; \
	fi

test-unit:
	GOCACHE=$(GO_CACHE) $(GO) test ./cmd/... ./internal/...

test-integration:
	GOCACHE=$(GO_CACHE) $(GO) test ./tests/integration/...

test-security:
	GOCACHE=$(GO_CACHE) $(GO) test ./tests/security/...

test-e2e:
	GOCACHE=$(GO_CACHE) $(GO) test ./tests/e2e/...

test-performance-local:
	GOCACHE=$(GO_CACHE) $(GO) test -bench=. ./cmd/... ./internal/... ./tests/performance/...

test: fmt-check test-unit test-integration test-security test-e2e

container-build:
	@if command -v $(DOCKER) >/dev/null 2>&1; then \
		$(DOCKER) build -f gateway/haproxy/Dockerfile -t $(IMAGE) .; \
	else \
		printf '%s\n' 'docker not installed; skipping container-build in local slice.'; \
	fi

container-scan:
	@if command -v $(TRIVY) >/dev/null 2>&1; then \
		$(TRIVY) image --exit-code 1 --severity HIGH,CRITICAL $(IMAGE); \
	else \
		printf '%s\n' 'trivy not installed; skipping container-scan in local slice.'; \
	fi

helm-lint:
	@if command -v $(HELM) >/dev/null 2>&1; then \
		$(HELM) lint charts/dev-connect-gateway; \
	else \
		printf '%s\n' 'helm not installed; skipping helm-lint in local slice.'; \
	fi

helm-template:
	@if command -v $(HELM) >/dev/null 2>&1; then \
		$(HELM) template dev-connect-gateway charts/dev-connect-gateway >/dev/null; \
	else \
		printf '%s\n' 'helm not installed; skipping helm-template in local slice.'; \
	fi

kustomize-build:
	@if command -v $(KUSTOMIZE) >/dev/null 2>&1; then \
		$(KUSTOMIZE) build kubernetes/base >/dev/null; \
		$(KUSTOMIZE) build kubernetes/overlays/autoscaling >/dev/null; \
	else \
		printf '%s\n' 'kustomize not installed; skipping kustomize-build in local slice.'; \
	fi

helm-package:
	@if command -v $(HELM) >/dev/null 2>&1; then \
		mkdir -p dist; \
		$(HELM) package charts/dev-connect-gateway -d dist; \
	else \
		printf '%s\n' 'helm not installed; skipping helm-package in local slice.'; \
	fi

sbom:
	@mkdir -p dist
	@if command -v $(SYFT) >/dev/null 2>&1; then \
		$(SYFT) dir:. -o spdx-json=dist/sbom.spdx.json; \
	else \
		printf '%s\n' 'syft not installed; skipping sbom in local slice.'; \
	fi

sign:
	@if command -v $(COSIGN) >/dev/null 2>&1; then \
		$(COSIGN) sign --yes $(IMAGE); \
	else \
		printf '%s\n' 'cosign not installed; skipping sign in local slice.'; \
	fi

release-check: fmt-check lint license-check test-unit test-integration test-security test-e2e helm-lint helm-template kustomize-build

build:
	GOCACHE=$(GO_CACHE) $(GO) build -buildvcs=false -o bin/dev-connect ./cmd/dev-connect

build-all:
	@mkdir -p dist
	@for target in \
		darwin/amd64 \
		darwin/arm64 \
		linux/amd64 \
		linux/arm64 \
		windows/amd64; do \
		goos=$${target%/*}; \
		goarch=$${target#*/}; \
		name=dev-connect-$${VERSION}-$${goos}-$${goarch}; \
		output=dist/$${name}; \
		if [ "$${goos}" = "windows" ]; then output=$${output}.exe; fi; \
		printf 'building %s/%s -> %s\n' "$${goos}" "$${goarch}" "$${output}"; \
		GOOS=$${goos} GOARCH=$${goarch} CGO_ENABLED=0 GOCACHE=$(GO_CACHE) $(GO) build -trimpath -buildvcs=false -ldflags="-s -w" -o "$${output}" ./cmd/dev-connect; \
	done

checksums:
	@if [ -d dist ]; then \
		(cd dist && shasum -a 256 * > checksums.txt); \
	else \
		printf '%s\n' 'dist directory not found; run make build-all or make helm-package first.'; \
		exit 1; \
	fi

clean:
	GOCACHE=$(GO_CACHE) $(GO) clean ./...
	rm -rf bin dist .cache
