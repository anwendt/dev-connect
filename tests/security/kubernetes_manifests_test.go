package security_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestKubernetesBaseManifestsAreValidYAML(t *testing.T) {
	for _, path := range kubernetesYAMLFiles(t, rootPath("deploy/kubernetes/base")) {
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
	for _, path := range kubernetesYAMLFiles(t, rootPath("deploy/kubernetes/base")) {
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
	content := readText(t, rootPath("deploy/kubernetes/base/service.yaml"))

	assertContains(t, content, "type: ClusterIP")
	assertContains(t, content, "port: 22")
	assertContains(t, content, "targetPort: 2222")
}

func TestGatewayDeploymentPodSecurity(t *testing.T) {
	content := readText(t, rootPath("deploy/kubernetes/base/deployment.yaml"))

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
		"readinessProbe:",
		"livenessProbe:",
		"tcpSocket:",
	} {
		assertContains(t, content, required)
	}

	assertNotContains(t, content, "NET_BIND_SERVICE")
	assertNotContains(t, content, "privileged: true")
}

func TestHAProxyConfigIsTCPOnlyAndNonPrivileged(t *testing.T) {
	content := readText(t, rootPath("deploy/kubernetes/base/configmap.yaml"))

	assertContains(t, content, "mode tcp")
	assertContains(t, content, "bind :2222")
	assertContains(t, content, "server dev01 dev01.example.internal:22 check")
	assertContains(t, content, "log stdout format raw local0")
	assertContains(t, content, "option tcplog")
	assertContains(t, content, "timeout client 12h")
	assertContains(t, content, "timeout server 12h")
	assertNotContains(t, content, "\n      bind :22\n")
	assertNotContains(t, content, "ssl")
	assertNotContains(t, content, "private_key")
}

func TestRBACLeastPrivilege(t *testing.T) {
	content := readText(t, rootPath("deploy/kubernetes/base/rbac.yaml"))

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
	content := readText(t, rootPath("deploy/kubernetes/base/networkpolicy.yaml"))

	assertContains(t, content, "policyTypes:")
	assertContains(t, content, "Ingress")
	assertContains(t, content, "Egress")
	assertContains(t, content, "port: 22")
	assertContains(t, content, "port: 53")
	assertContains(t, content, "protocol: UDP")
	assertNotContains(t, content, "0.0.0.0/0")
}

func TestHPAIsProvidedButDisabledByDefaultForKustomizeBase(t *testing.T) {
	base := readText(t, rootPath("deploy/kubernetes/base/kustomization.yaml"))
	overlay := readText(t, rootPath("deploy/kubernetes/overlays/autoscaling/kustomization.yaml"))
	hpa := readText(t, rootPath("deploy/kubernetes/overlays/autoscaling/hpa.yaml"))

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
	assertContains(t, deployment, "readinessProbe:")
	assertContains(t, deployment, "livenessProbe:")
	assertContains(t, deployment, "tcpSocket:")
	assertContains(t, service, "type: ClusterIP")
	assertContains(t, service, "targetPort: 2222")
	assertContains(t, hpa, "if .Values.autoscaling.enabled")
	assertNotContains(t, service, "LoadBalancer")
	assertNotContains(t, service, "NodePort")
}

func TestHelmChartPreservesTCPLoggingDefaults(t *testing.T) {
	configmap := readText(t, rootPath("charts/dev-connect-gateway/templates/configmap.yaml"))

	assertContains(t, configmap, "mode tcp")
	assertContains(t, configmap, "log stdout format raw local0")
	assertContains(t, configmap, "option tcplog")
	assertContains(t, configmap, "timeout client 12h")
	assertContains(t, configmap, "timeout server 12h")
	assertNotContains(t, configmap, "ssl")
	assertNotContains(t, configmap, "private_key")
}

func TestHelmChartIncludesRancherMonitoringIntegration(t *testing.T) {
	values := readText(t, rootPath("charts/dev-connect-gateway/values.yaml"))
	deployment := readText(t, rootPath("charts/dev-connect-gateway/templates/deployment.yaml"))
	service := readText(t, rootPath("charts/dev-connect-gateway/templates/service.yaml"))
	configmap := readText(t, rootPath("charts/dev-connect-gateway/templates/configmap.yaml"))
	serviceMonitor := readText(t, rootPath("charts/dev-connect-gateway/templates/servicemonitor.yaml"))
	prometheusRule := readText(t, rootPath("charts/dev-connect-gateway/templates/prometheusrule.yaml"))
	dashboard := readText(t, rootPath("charts/dev-connect-gateway/templates/grafana-dashboard.yaml"))

	for _, required := range []string{
		"monitoring:",
		"metrics:",
		"serviceMonitor:",
		"prometheusRule:",
		"grafanaDashboard:",
		"targetLabel: dev_connect_target",
		"namespace: cattle-dashboards",
		"grafana_dashboard: \"1\"",
		"release: rancher-monitoring",
	} {
		assertContains(t, values, required)
	}
	assertContains(t, configmap, "prometheus-exporter")
	assertContains(t, deployment, "containerPort: {{ .Values.monitoring.metrics.port }}")
	assertContains(t, service, "targetPort: {{ .Values.monitoring.metrics.port }}")
	assertContains(t, serviceMonitor, "kind: ServiceMonitor")
	assertContains(t, serviceMonitor, "monitoring.coreos.com/v1")
	assertContains(t, serviceMonitor, "metricRelabelings:")
	assertContains(t, prometheusRule, "kind: PrometheusRule")
	assertContains(t, prometheusRule, "DevConnectGatewayBackendDown")
	assertContains(t, prometheusRule, "dev_connect_target")
	assertContains(t, dashboard, "kind: ConfigMap")
	assertContains(t, dashboard, "dev-connect Gateway")
	assertContains(t, dashboard, "\"name\": \"target\"")
	assertContains(t, dashboard, "\"label\": \"Server\"")
	assertContains(t, dashboard, "\"options\": [")
	assertContains(t, dashboard, "\"text\": \"{{ .Values.target.name }}\"")
	assertContains(t, dashboard, "label_values(haproxy_backend_status")
	assertContains(t, dashboard, "dev_connect_target=\\\"$target\\\"")
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
