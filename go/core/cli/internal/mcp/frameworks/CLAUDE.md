# cli/internal/mcp/frameworks/

Language-specific MCP server project generators. Each framework implements `GenerateProject(config)` using embedded Go templates.

`frameworks.go` provides the factory/registry for selecting a generator by language name.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `common/` | `BaseGenerator` shared logic for template rendering, directory creation, and tool file generation |
| `golang/` | Go MCP server generator (uses `go mod tidy` post-generation) |
| `java/` | Java MCP server generator (Maven/Spring Boot structure) |
| `python/` | Python MCP server generator (pyproject.toml + src layout) |
| `typescript/` | TypeScript MCP server generator (npm + src layout) |
