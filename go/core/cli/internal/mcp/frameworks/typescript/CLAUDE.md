# cli/internal/mcp/frameworks/typescript/

TypeScript MCP server project generator. Scaffolds an npm project with src/tests layout.

`generator.go` implements `GenerateProject` using embedded templates.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Embedded templates: `src/tools/` (tool implementations), `src/types/` (type definitions), `tests/` (test stubs), package.json, tsconfig.json, Dockerfile |
