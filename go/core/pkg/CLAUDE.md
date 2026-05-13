# go/core/pkg — Public Packages

Exported packages usable outside `internal/`. These provide shared application-level utilities.

## Packages

| Package | Role |
|---------|------|
| `app/` | Application bootstrap (wires together controller, HTTP server, database) |
| `auth/` | Authentication utilities shared across components |
| `env/` | Environment variable helpers |
| `mcp/` | MCP client/server utilities for external use |
| `migrations/` | Database migration definitions and runner |
| `sandboxbackend/` | Sandbox agent backend (ephemeral execution environment) |
| `translator/` | Public translator interface (used by CLI and tests) |
