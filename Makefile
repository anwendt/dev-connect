.PHONY: help fmt fmt-check lint license-check test-unit test-integration test-security test-e2e test-performance-local test build build-all package-windows-bundle checksums container-build container-push container-scan helm-lint helm-template helm-package helm-push kustomize-build sbom sign release-check clean

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
CONTAINER_REGISTRY_HOST ?= ghcr.io
CONTAINER_REGISTRY_USERNAME ?= $(GITHUB_ACTOR)
CONTAINER_REGISTRY_PASSWORD ?= $(GITHUB_TOKEN)
HELM_REGISTRY ?= oci://ghcr.io/anwendt/charts
HELM_REGISTRY_HOST ?= ghcr.io
HELM_REGISTRY_USERNAME ?= $(GITHUB_ACTOR)
HELM_REGISTRY_PASSWORD ?= $(GITHUB_TOKEN)
HELM_CHART_NAME ?= dev-connect-gateway
HELM_CHART_VERSION ?= $(shell awk -F': *' '/^version:/ { gsub(/"/, "", $$2); print $$2 }' charts/dev-connect-gateway/Chart.yaml)
VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || printf 'unknown')
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS ?= -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)
KUBECTL_VERSION ?= v1.30.14
KUBECTL_WINDOWS_AMD64_URL ?= https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/windows/amd64/kubectl.exe

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
		'  make container-push   Push gateway container image' \
		'  make helm-lint        Run Helm chart linting' \
		'  make helm-template    Render Helm chart templates' \
		'  make helm-package     Package Helm chart into dist/' \
		'  make helm-push        Push packaged Helm chart as OCI artifact' \
		'  make kustomize-build  Render Kustomize base and autoscaling overlay' \
		'  make sbom             Generate SPDX SBOM into dist/' \
		'  make sign             Sign container image when cosign is available' \
		'  make release-check    Validate release readiness' \
		'  make build            Build dev-connect' \
		'  make build-all        Build release binaries for Windows, Linux, and macOS' \
		'  make package-windows-bundle Build Windows zip with dev-connect.exe and kubectl.exe' \
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

container-push: container-build
	@if ! command -v $(DOCKER) >/dev/null 2>&1; then \
		printf '%s\n' 'docker not installed; cannot push container image.'; \
		exit 1; \
	fi
	@if [ -z "$(CONTAINER_REGISTRY_USERNAME)" ] || [ -z "$(CONTAINER_REGISTRY_PASSWORD)" ]; then \
		printf '%s\n' 'CONTAINER_REGISTRY_USERNAME and CONTAINER_REGISTRY_PASSWORD are required for container-push.'; \
		exit 1; \
	fi
	@printf '%s' "$(CONTAINER_REGISTRY_PASSWORD)" | $(DOCKER) login $(CONTAINER_REGISTRY_HOST) --username "$(CONTAINER_REGISTRY_USERNAME)" --password-stdin
	$(DOCKER) push $(IMAGE)

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
		$(KUSTOMIZE) build deploy/kubernetes/base >/dev/null; \
		$(KUSTOMIZE) build deploy/kubernetes/overlays/autoscaling >/dev/null; \
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

helm-push: helm-package
	@if ! command -v $(HELM) >/dev/null 2>&1; then \
		printf '%s\n' 'helm not installed; cannot push Helm OCI chart.'; \
		exit 1; \
	fi
	@if [ -z "$(HELM_REGISTRY_USERNAME)" ] || [ -z "$(HELM_REGISTRY_PASSWORD)" ]; then \
		printf '%s\n' 'HELM_REGISTRY_USERNAME and HELM_REGISTRY_PASSWORD are required for helm-push.'; \
		exit 1; \
	fi
	@printf '%s' "$(HELM_REGISTRY_PASSWORD)" | $(HELM) registry login $(HELM_REGISTRY_HOST) --username "$(HELM_REGISTRY_USERNAME)" --password-stdin
	@if $(HELM) show chart "$(HELM_REGISTRY)/$(HELM_CHART_NAME)" --version "$(HELM_CHART_VERSION)" >/dev/null 2>&1; then \
		printf 'Helm chart %s/%s version %s already exists; skipping helm-push.\n' "$(HELM_REGISTRY)" "$(HELM_CHART_NAME)" "$(HELM_CHART_VERSION)"; \
		exit 0; \
	fi
	@chart="$$(ls dist/dev-connect-gateway-*.tgz | head -n 1)"; \
	if [ -z "$$chart" ]; then \
		printf '%s\n' 'no packaged Helm chart found in dist/'; \
		exit 1; \
	fi; \
	$(HELM) push "$$chart" $(HELM_REGISTRY)

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
	GOCACHE=$(GO_CACHE) $(GO) build -buildvcs=false -ldflags="$(LDFLAGS)" -o bin/dev-connect ./cmd/dev-connect

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
		GOOS=$${goos} GOARCH=$${goarch} CGO_ENABLED=0 GOCACHE=$(GO_CACHE) $(GO) build -trimpath -buildvcs=false -ldflags="$(LDFLAGS)" -o "$${output}" ./cmd/dev-connect; \
	done

package-windows-bundle: build-all
	@if ! command -v curl >/dev/null 2>&1; then \
		printf '%s\n' 'curl is required to download kubectl.exe for the Windows bundle.'; \
		exit 1; \
	fi
	@if ! command -v zip >/dev/null 2>&1; then \
		printf '%s\n' 'zip is required to package the Windows bundle.'; \
		exit 1; \
	fi
	@mkdir -p dist/windows-amd64-bundle
	cp dist/dev-connect-$(VERSION)-windows-amd64.exe dist/windows-amd64-bundle/dev-connect.exe
	curl -fsSLo dist/windows-amd64-bundle/kubectl.exe "$(KUBECTL_WINDOWS_AMD64_URL)"
	cp LICENSE dist/windows-amd64-bundle/LICENSE
	cp README.md dist/windows-amd64-bundle/README.md
	(cd dist/windows-amd64-bundle && zip -q ../dev-connect-$(VERSION)-windows-amd64-bundle.zip dev-connect.exe kubectl.exe LICENSE README.md)
	rm -rf dist/windows-amd64-bundle

checksums:
	@if [ -d dist ]; then \
		(cd dist && find . -maxdepth 1 -type f ! -name checksums.txt -print | sed 's#^\./##' | sort | xargs shasum -a 256 > checksums.txt); \
	else \
		printf '%s\n' 'dist directory not found; run make build-all or make helm-package first.'; \
		exit 1; \
	fi

clean:
	GOCACHE=$(GO_CACHE) $(GO) clean ./...
	rm -rf bin dist .cache
