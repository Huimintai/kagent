# go/core/cli — CLI Tool

The `kagent` CLI for interacting with the platform from the terminal.

## Structure

| Directory | Role |
|-----------|------|
| `cmd/kagent/` | Main entry point for the CLI binary |
| `internal/` | Internal implementation packages |
| `test/` | CLI test utilities |

## internal/ Packages

| Package | Role |
|---------|------|
| `agent/` | Agent-related commands |
| `cli/` | CLI framework and root command |
| `common/` | Shared helpers |
| `config/` | CLI configuration management |
| `mcp/` | MCP server commands |
| `profiles/` | Profile management (contexts, clusters) |
| `tui/` | Terminal UI components |

## Testing

See `TESTING.md` in this directory for CLI test conventions.
