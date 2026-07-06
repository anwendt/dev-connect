package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestKubernetesBaseManifestsAreValidYAML(t *testing.T) {
	for _, path := range kubernetesYAMLFiles(t, rootPath("kubernetes/base")) {
		for _, document := range yamlDocuments(t, path) {
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(document), &parsed); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			if parsed["kind"] == "" {
				t.Fatalf("%s missing kind", path)
			}
			if parsed["apiVersion"] == "" {
				t.Fatalf("%s missing apiVersion", path)
			}
		}
	}
}

func TestNoPublicKubernetesExposure(t *testing.T) {
	for _, path := range kubernetesYAMLFiles(t, rootPath("kubernetes/base")) {
		content := readText(t, path)
		for _, forbidden := range []string{
			"kind: Ingress",
			"type: LoadBalancer",
			"type: NodePort",
		} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s contains forbidden exposure %q", path, forbidden)
			}
		}
	}
}

func TestGatewayServicePortMapping(t *testing.T) {
	content := readText(t, rootPath("kubernetes/base/service.yaml"))

	assertContains(t, content, "type: ClusterIP")
	assertContains(t, content, "port: 22")
	assertContains(t, content, "targetPort: 2222")
}

func TestGatewayDeploymentPodSecurity(t *testing.T) {
	content := readText(t, rootPath("kubernetes/base/deployment.yaml"))

	for _, required := range []string{
		"containerPort: 2222",
		"runAsNonRoot: true",
		"allowPrivilegeEscalation: false",
		"readOnlyRootFilesystem: true",
		"seccompProfile:",
		"type: RuntimeDefault",
		"drop:",
		"- ALL",
		"automountServiceAccountToken: false",
		"cpu: 100m",
		"memory: 128Mi",
		"cpu: 500m",
		"memory: 256Mi",
	} {
		assertContains(t, content, required)
	}

	assertNotContains(t, content, "NET_BIND_SERVICE")
	assertNotContains(t, content, "privileged: true")
}

func TestHAProxyConfigIsTCPOnlyAndNonPrivileged(t *testing.T) {
	content := readText(t, rootPath("kubernetes/base/configmap.yaml"))

	assertContains(t, content, "mode tcp")
	assertContains(t, content, "bind :2222")
	assertContains(t, content, "server dev01 dev01.example.internal:22 check")
	assertNotContains(t, content, "\n      bind :22\n")
	assertNotContains(t, content, "ssl")
	assertNotContains(t, content, "private_key")
}

func TestRBACLeastPrivilege(t *testing.T) {
	content := readText(t, rootPath("kubernetes/base/rbac.yaml"))

	assertContains(t, content, "resources:")
	assertContains(t, content, "pods/portforward")
	assertContains(t, content, "verbs:")
	assertContains(t, content, "create")

	for _, forbidden := range []string{
		"cluster-admin",
		"ClusterRoleBinding",
		"resources: [\"*\"]",
		"verbs: [\"*\"]",
		"secrets",
		"deployments",
		"configmaps",
		"pods/exec",
	} {
		assertNotContains(t, strings.ToLower(content), strings.ToLower(forbidden))
	}
}

func TestNetworkPolicyRestrictsEgressToSSHAndConditionalDNS(t *testing.T) {
	content := readText(t, rootPath("kubernetes/base/networkpolicy.yaml"))

	assertContains(t, content, "policyTypes:")
	assertContains(t, content, "Ingress")
	assertContains(t, content, "Egress")
	assertContains(t, content, "port: 22")
	assertContains(t, content, "port: 53")
	assertContains(t, content, "protocol: UDP")
	assertNotContains(t, content, "0.0.0.0/0")
}

func TestHPAIsProvidedButDisabledByDefaultForKustomizeBase(t *testing.T) {
	base := readText(t, rootPath("kubernetes/base/kustomization.yaml"))
	overlay := readText(t, rootPath("kubernetes/overlays/autoscaling/kustomization.yaml"))
	hpa := readText(t, rootPath("kubernetes/overlays/autoscaling/hpa.yaml"))

	assertNotContains(t, base, "hpa.yaml")
	assertContains(t, overlay, "hpa.yaml")
	assertContains(t, hpa, "kind: HorizontalPodAutoscaler")
}

func TestHelmChartPreservesSecurityDefaults(t *testing.T) {
	values := readText(t, rootPath("charts/dev-connect-gateway/values.yaml"))
	deployment := readText(t, rootPath("charts/dev-connect-gateway/templates/deployment.yaml"))
	service := readText(t, rootPath("charts/dev-connect-gateway/templates/service.yaml"))
	hpa := readText(t, rootPath("charts/dev-connect-gateway/templates/hpa.yaml"))

	assertContains(t, values, "enabled: false")
	assertContains(t, deployment, "automountServiceAccountToken: false")
	assertContains(t, deployment, "containerPort: 2222")
	assertContains(t, service, "type: ClusterIP")
	assertContains(t, service, "targetPort: 2222")
	assertContains(t, hpa, "if .Values.autoscaling.enabled")
	assertNotContains(t, service, "LoadBalancer")
	assertNotContains(t, service, "NodePort")
}

func kubernetesYAMLFiles(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}

	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func yamlDocuments(t *testing.T, path string) []string {
	t.Helper()

	content := readText(t, path)
	parts := strings.Split(content, "\n---")
	documents := []string{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			documents = append(documents, trimmed)
		}
	}
	return documents
}

func readText(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, data, want string) {
	t.Helper()

	if !strings.Contains(data, want) {
		t.Fatalf("data missing %q:\n%s", want, data)
	}
}

func assertNotContains(t *testing.T, data, forbidden string) {
	t.Helper()

	if strings.Contains(data, forbidden) {
		t.Fatalf("data contains forbidden %q:\n%s", forbidden, data)
	}
}

func rootPath(path string) string {
	return filepath.Join("..", "..", path)
}
