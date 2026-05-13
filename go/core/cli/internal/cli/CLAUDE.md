# go/core/cli/internal/cli — CLI Command Implementations

Cobra command implementations for the `kagent` CLI tool, organized by domain.

## Sub-packages

| Package | Role |
|---------|------|
| `agent/` | Agent lifecycle commands (init, build, run, deploy, get, invoke, add-mcp, dashboard, bug-report) |
| `envdoc/` | Environment variable documentation generator |
| `mcp/` | MCP server lifecycle commands (init, build, run, deploy, add-tool, inspector, secrets) |
| `platform/` | Platform commands (connect, OAuth login) |

## Module Boundary

- **Imports**: `cli/internal/agent`, `cli/internal/mcp`, `cli/internal/common`, `cli/internal/tui`
- **Imported by**: `cli/cmd/kagent/` (root command wiring)
