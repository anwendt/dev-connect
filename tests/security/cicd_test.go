package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestMakefileContainsApprovedCICDTargets(t *testing.T) {
	makefile := readText(t, rootPath("Makefile"))

	for _, target := range []string{
		"fmt-check:",
		"lint:",
		"license-check:",
		"test-unit:",
		"test-integration:",
		"test-security:",
		"test-e2e:",
		"test-performance-local:",
		"container-build:",
		"container-push:",
		"container-scan:",
		"helm-lint:",
		"helm-template:",
		"helm-package:",
		"helm-push:",
		"sbom:",
		"sign:",
		"build-all:",
		"checksums:",
		"release-check:",
	} {
		assertContains(t, makefile, target)
	}
}

func TestWorkflowsAreValidYAMLAndUseMakeTargets(t *testing.T) {
	workflowDir := rootPath(".github/workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		t.Fatalf("read workflows: %v", err)
	}

	workflowCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}
		workflowCount++
		path := filepath.Join(workflowDir, entry.Name())
		content := readText(t, path)

		var parsed map[string]any
		if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
			t.Fatalf("workflow %s is invalid YAML: %v", path, err)
		}
		assertContains(t, content, "make ")
		assertNotContains(t, content, "go test ./")
		assertNotContains(t, content, "helm lint ")
		assertNotContains(t, content, "helm package ")
		assertNotContains(t, content, "docker build ")
	}

	if workflowCount < 4 {
		t.Fatalf("workflow count = %d, want at least 4", workflowCount)
	}
}

func TestWorkflowPermissionsAreLeastPrivilege(t *testing.T) {
	ci := readText(t, rootPath(".github/workflows/ci.yml"))
	security := readText(t, rootPath(".github/workflows/security-scan.yml"))
	performance := readText(t, rootPath(".github/workflows/performance.yml"))
	release := readText(t, rootPath(".github/workflows/release.yml"))

	for name, content := range map[string]string{
		"ci":          ci,
		"security":    security,
		"performance": performance,
	} {
		assertContains(t, content, "permissions:")
		assertContains(t, content, "contents: read")
		assertNotContains(t, content, "contents: write")
		assertNotContains(t, content, "packages: write")
		assertNotContains(t, content, "id-token: write")
		t.Logf("checked %s workflow permissions", name)
	}

	assertContains(t, release, "contents: write")
	assertContains(t, release, "packages: write")
	assertContains(t, release, "id-token: write")
	assertContains(t, release, "attestations: write")
}

func TestReleaseWorkflowIncludesSupplyChainControls(t *testing.T) {
	release := readText(t, rootPath(".github/workflows/release.yml"))

	for _, required := range []string{
		"make release-check",
		"make build-all",
		"make sbom",
		"make container-build",
		"make container-scan",
		"make container-push",
		"make helm-package",
		"make helm-push",
		"make checksums",
		"make sign",
		"cosign-installer",
		"anchore/sbom-action",
		"aquasecurity/setup-trivy",
		"go install github.com/google/go-licenses@latest",
		"actions/attest-build-provenance",
		"CONTAINER_REGISTRY_PASSWORD",
		"HELM_REGISTRY_PASSWORD",
		"gh release create",
		"gh release upload",
	} {
		assertContains(t, release, required)
	}
}

func TestCIWorkflowValidatesSupportedClientOperatingSystems(t *testing.T) {
	ci := readText(t, rootPath(".github/workflows/ci.yml"))

	for _, required := range []string{
		"ubuntu-latest",
		"macos-15",
		"windows-latest",
		"make test-unit",
		"make test-e2e",
		"make build",
	} {
		assertContains(t, ci, required)
	}
}

func TestMakefileBuildAllTargetsSupportedReleasePlatforms(t *testing.T) {
	makefile := readText(t, rootPath("Makefile"))

	for _, required := range []string{
		"darwin/amd64",
		"darwin/arm64",
		"linux/amd64",
		"linux/arm64",
		"windows/amd64",
		"CGO_ENABLED=0",
	} {
		assertContains(t, makefile, required)
	}
}

func TestRenovateCoversApprovedDependencyTypes(t *testing.T) {
	renovate := readText(t, rootPath("renovate.json"))

	for _, required := range []string{
		"gomod",
		"github-actions",
		"dockerfile",
	} {
		assertContains(t, renovate, required)
	}
}

func TestNoGeneratedArtifactsAreCommitted(t *testing.T) {
	gitignore := readText(t, rootPath(".gitignore"))
	for _, ignored := range []string{
		"bin/",
		"dist/",
		".cache/",
		"sbom.spdx.json",
	} {
		assertContains(t, gitignore, ignored)
	}
}
