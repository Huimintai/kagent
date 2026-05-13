# go/core/internal — Internal Packages

All internal implementation packages for the core infrastructure.

## Packages

| Package | Role |
|---------|------|
| `controller/` | K8s controllers — reconcile CRDs into Deployments, ConfigMaps, Services |
| `httpserver/` | REST API — handlers, auth, middleware, error handling |
| `database/` | PostgreSQL data layer — sqlc queries, generated code, fake client |
| `a2a/` | Agent-to-Agent protocol server |
| `aicoreproxy/` | Proxy to SAP AI Core (LLM gateway) |
| `broker/` | Credential broker — manages platform credentials |
| `goruntime/` | Go runtime executor for in-process agents |
| `mcp/` | MCP server integration |
| `metrics/` | Prometheus metric collectors |
| `telemetry/` | OpenTelemetry setup and spans |
| `utils/` | Shared internal utilities |
| `version/` | Build version information |
| `dbtest/` | Database test helpers |
