# go/ — Go Workspace

Single Go module (`github.com/kagent-dev/kagent/go`) containing three logical packages: api, core, and adk.

## Module Boundary Diagram

```
         ┌───────────┐
         │  go/api   │  (shared types)
         └─────┬─────┘
           ┌───┴───┐
           │       │
     ┌─────▼──┐ ┌──▼─────┐
     │go/core │ │ go/adk │
     └────────┘ └────────┘
```

## Strict Import Rules

| Package | May Import | NEVER Imports |
|---------|-----------|---------------|
| `go/api` | stdlib, external deps | core, adk |
| `go/core` | `go/api`, stdlib, external deps | adk |
| `go/adk` | `go/api`, stdlib, external deps | core |

## API Versioning

- **v1alpha2** — current, all new work goes here
- **v1alpha1** — deprecated, do not modify unless critical bug

## Quick Commands

```bash
cd go
go test ./...              # run all tests
golangci-lint run ./...    # lint
go generate ./...          # regenerate CRD code
```
