# cli/internal/

Internal packages for the `kagent` CLI tool.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `agent/` | Agent project scaffolding (`kagent agent init`) |
| `cli/` | Cobra command definitions organized by domain (agent, mcp, platform, envdoc) |
| `common/` | Shared utilities: exec, fs, generator, image, k8s, prompt helpers |
| `config/` | CLI configuration types and utilities |
| `mcp/` | MCP server project scaffolding (`kagent mcp init`) |
| `profiles/` | Predefined agent profile YAML files for quick-start |
| `tui/` | Terminal UI (Bubble Tea) for interactive chat workspace |
