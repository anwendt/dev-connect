# dev-connect-gateway Helm Chart

This chart deploys one HAProxy-based TCP gateway for a single dev-connect
development target.

## OCI Installation

Release builds publish this chart as an OCI artifact in GHCR.

Example:

```sh
helm install dev-connect-dev01 oci://ghcr.io/anwendt/charts/dev-connect-gateway \
  --version 0.2.5 \
  --namespace dev-connect \
  --create-namespace
```

## Rancher Monitoring

Rancher Monitoring is supported through the Prometheus Operator resources used
by the `rancher-monitoring` chart:

- HAProxy Prometheus metrics on an internal metrics port.
- `ServiceMonitor` for Prometheus scraping.
- `PrometheusRule` for gateway/backend alerts.
- Grafana dashboard `ConfigMap` for Rancher persistent dashboards.

Monitoring is disabled by default so the chart can still be installed on
clusters where the Rancher Monitoring CRDs are not present.

Enable it for Rancher Monitoring:

```sh
helm upgrade --install dev-connect-dev01 charts/dev-connect-gateway \
  --namespace dev-connect \
  --create-namespace \
  --set target.name=dev01 \
  --set target.host=172.28.192.14 \
  --set networkPolicy.backendCIDR=172.28.192.14/32 \
  --set networkPolicy.dnsEgress.enabled=false \
  --set monitoring.enabled=true
```

The dashboard is installed into `cattle-dashboards` by default with the label
`grafana_dashboard: "1"`, which is the Rancher-supported persistent dashboard
mechanism. If the Rancher Monitoring chart is configured to watch another
dashboard namespace or label, override:

```yaml
monitoring:
  grafanaDashboard:
    namespace: cattle-dashboards
    labels:
      grafana_dashboard: "1"
  serviceMonitor:
    labels:
      release: rancher-monitoring
  prometheusRule:
    labels:
      release: rancher-monitoring
```

No public endpoint is created. The metrics port is exposed only through the
ClusterIP Service and NetworkPolicy ingress when `monitoring.enabled=true`.
