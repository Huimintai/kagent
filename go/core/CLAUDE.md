# go/core — Infrastructure Layer

Kubernetes controllers, HTTP server, database layer, and CLI. This is the platform's control plane.

## Sub-packages

| Package | Role |
|---------|------|
| `internal/controller/` | K8s operator controllers (reconcile CRDs into runtime resources) |
| `internal/httpserver/` | REST API server (handlers, auth, middleware) |
| `internal/database/` | PostgreSQL + pgvector data layer (sqlc-generated) |
| `internal/a2a/` | Agent-to-Agent protocol implementation |
| `internal/aicoreproxy/` | SAP AI Core proxy |
| `internal/broker/` | Credential broker |
| `internal/goruntime/` | Go runtime executor |
| `internal/mcp/` | MCP integration |
| `internal/metrics/` | Prometheus metrics |
| `internal/telemetry/` | OpenTelemetry |
| `internal/utils/` | Utilities |
| `internal/version/` | Version info |
| `internal/dbtest/` | Database test helpers |
| `pkg/` | Public packages (app, auth, env, mcp, migrations, sandboxbackend, translator) |
| `cli/` | CLI tool (`kagent`) |
| `cmd/` | Binary entry points (controller, aicore-proxy) |
| `test/` | E2E tests |

## Module Boundary

Imports `go/api` only. NEVER imports `go/adk`.
