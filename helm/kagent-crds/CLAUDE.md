# helm/kagent-crds/ - CRD Chart

Custom Resource Definition chart. Must be installed before the main kagent chart.

## Structure

| Directory | Purpose |
|-----------|---------|
| `templates/` | CRD YAML manifests |

## CRDs

| File | Resource |
|------|----------|
| `kagent.dev_agents.yaml` | Agent |
| `kagent.dev_memories.yaml` | Memory |
| `kagent.dev_modelconfigs.yaml` | ModelConfig |
| `kagent.dev_modelproviderconfigs.yaml` | ModelProviderConfig |
| `kagent.dev_platformcredentials.yaml` | PlatformCredential |
| `kagent.dev_remotemcpservers.yaml` | RemoteMCPServer |
| `kagent.dev_sandboxagents.yaml` | SandboxAgent |
| `kagent.dev_scheduledruns.yaml` | ScheduledRun |
| `kagent.dev_toolservers.yaml` | ToolServer |
