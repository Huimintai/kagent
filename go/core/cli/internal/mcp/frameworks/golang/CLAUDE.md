# cli/internal/mcp/frameworks/golang/

Go MCP server project generator. Scaffolds a Go module with cmd/server entry point and internal/tools package.

`generator.go` extends `BaseGenerator`, runs `go mod tidy` after template rendering.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Embedded Go templates for the generated project: `cmd/server/` (main.go), `internal/tools/` (tool implementations), go.mod, Dockerfile, etc. |
