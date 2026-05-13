# helm/agents/ - Pre-built Agent Charts

Helm charts for deploying ready-made infrastructure and monitoring agents.

## Charts

| Directory | Agent |
|-----------|-------|
| `argo-rollouts/` | Argo Rollouts management agent |
| `cilium-debug/` | Cilium debugging agent |
| `cilium-manager/` | Cilium management agent |
| `cilium-policy/` | Cilium network policy agent |
| `helm/` | Helm operations agent |
| `istio/` | Istio service mesh agent |
| `k8s/` | General Kubernetes agent |
| `kgateway/` | Kubernetes Gateway API agent |
| `observability/` | Observability/monitoring agent |
| `promql/` | PromQL query agent |

Each chart contains: `Chart.yaml`, `Chart-template.yaml`, `values.yaml`, `templates/`.
