# go/api — Shared Types

Shared type definitions imported by both `go/core` and `go/adk`. This package defines the contract between all Go components.

## Sub-packages

| Package | Role | Key Contents |
|---------|------|--------------|
| `v1alpha2/` | CRD type definitions | Agent, ModelConfig, ToolServer, RemoteMCPServer, PlatformCredential, SandboxAgent, ScheduledRun |
| `client/` | HTTP client library | Typed client for platform REST API (agents, sessions, tools, models, etc.) |
| `database/` | Database models | Shared DB model structs and client interface |
| `httpapi/` | HTTP API types | Request/response structs for REST endpoints |
| `adk/` | ADK configuration types | Agent config structures passed to ADK runtime |
| `config/` | Build configuration | CRD base manifests for code generation |
| `utils/` | Utility types | Shared helper types |
| `hack/` | Code generation helpers | Boilerplate templates for generated files |

## Module Boundary

Imported by `go/core` and `go/adk`. NEVER imports either of them.
