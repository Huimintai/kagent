# go/core/internal/controller — K8s Controllers

Kubernetes operator controllers that reconcile CRDs into runtime resources (Deployments, ConfigMaps, Services).

## Controllers

| File | Reconciles | Description |
|------|-----------|-------------|
| `agent_controller.go` | `Agent` | Deploys agent pods with translated config |
| `modelconfig_controller.go` | `ModelConfig` | Manages model configuration secrets |
| `modelproviderconfig_controller.go` | `ModelProviderConfig` | Manages provider credential secrets |
| `service_controller.go` | `Agent` (Service) | Creates/updates K8s Services for agents |
| `remote_mcp_server_controller.go` | `RemoteMCPServer` | Reconciles remote MCP server connections |
| `mcp_server_tool_controller.go` | `RemoteMCPServer` (tools) | Discovers tools from MCP servers |
| `platformcredential_controller.go` | `PlatformCredential` | Manages platform-level credentials |
| `sandboxagent_controller.go` | `SandboxAgent` | Deploys sandbox (ephemeral) agents |
| `scheduledrun_controller.go` | `ScheduledRun` | Manages cron-scheduled agent runs |
| `scheduledrun_scheduler.go` | — | Cron scheduling logic for ScheduledRun |

## Sub-packages

| Package | Role |
|---------|------|
| `reconciler/` | Generic reconciler framework (status updates, finalizers) |
| `translator/` | CRD-to-ADK config translation (agent specs to runtime manifests) |
| `predicates/` | Event filter predicates (e.g., discovery disabled) |
| `provider/` | Tool provider discovery |
| `hack/` | Code generation boilerplate |

## Helper Files

| File | Role |
|------|------|
| `agentobject_helpers.go` | Shared helpers for Agent/SandboxAgent objects |
| `watch_helpers.go` | Watch setup helpers for dependent resources |
