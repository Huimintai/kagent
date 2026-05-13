# cli/internal/mcp/frameworks/java/

Java MCP server project generator. Scaffolds a Maven project with Spring Boot structure.

`generator.go` implements `GenerateProject` using embedded templates. Includes unit tests in `generator_test.go`.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Embedded templates for Maven pom.xml, `src/main/java/com/example/` (server + tools), `src/test/java/com/example/` (test stubs) |
