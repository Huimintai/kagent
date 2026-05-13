# CLAUDE.md - Kagent Development Guide

This document provides essential guidance for AI agents working in the kagent repository.

---

## Development Workflow Skill

**For detailed development workflows, use the `kagent-dev` skill.** The skill provides comprehensive guidance on:

- Adding CRD fields (step-by-step with examples)
- Running and debugging E2E tests
- PR review workflows
- Local development setup
- CI failure troubleshooting
- Common development patterns

The skill includes detailed reference materials on CRD workflows, translator patterns, E2E debugging, and CI failures.

---

## Project Overview

**Kagent** is a Kubernetes-native framework for building, deploying, and managing AI agents.

**Architecture:**
```
┌─────────────┐   ┌──────────────┐   ┌─────────────┐
│ Controller  │   │  HTTP Server │   │     UI      │
│    (Go)     │──▶│   (Go)       │──▶│ (Next.js)   │
└─────────────┘   └──────────────┘   └─────────────┘
       │                  │
       ▼                  ▼
┌─────────────┐   ┌──────────────┐
│  Database   │   │ Agent Runtime│
│ (SQLite/PG) │   │   (Python)   │
└─────────────┘   └──────────────┘
```

**Current Version:** v0.x.x (Alpha stage)

---

## Repository Structure

```
kagent/
├── go/                          # Go module (single go.mod)
│   ├── api/                     # Shared types: CRDs, ADK types, DB models, HTTP client
│   │   └── v1alpha2/            # Current CRD version (8 CRDs)
│   ├── core/                    # Infrastructure: controllers, HTTP server, CLI
│   │   ├── internal/controller/ # 10 controllers + translators
│   │   ├── internal/httpserver/ # REST API (40+ routes)
│   │   ├── cli/                 # kagent CLI tool
│   │   └── test/e2e/            # E2E tests
│   └── adk/                     # Go Agent Development Kit
├── python/                      # Agent runtime and ADK
│   ├── packages/                # UV workspace (9 packages)
│   │   ├── kagent-adk/          # Main Python ADK
│   │   ├── kagent-core/         # Core utilities
│   │   ├── kagent-skills/       # Skills framework
│   │   ├── kagent-sota-adapter/ # SOTA adapter (Claude Code, Codex)
│   │   ├── kagent-crewai/       # CrewAI integration
│   │   ├── kagent-langgraph/    # LangGraph integration
│   │   └── kagent-openai/       # OpenAI utilities
│   └── samples/                 # Example agents
├── ui/                          # Next.js web interface
├── helm/                        # Kubernetes deployment charts
│   ├── kagent-crds/             # CRD chart (install first)
│   └── kagent/                  # Main application chart
├── scripts/build/               # Per-component build scripts
├── docker/                      # Dockerfiles for non-Go components
└── docs/architecture/           # Architecture documentation
```

---

## CRDs (v1alpha2)

| CRD | Purpose |
|-----|---------|
| Agent | Main AI agent resource |
| SandboxAgent | Sandbox-based agent variant |
| AgentHarness | Multi-channel agent harnesses |
| ModelConfig | LLM provider configuration |
| ModelProviderConfig | Model provider setup |
| PlatformCredential | Platform credentials |
| RemoteMCPServer | External MCP tool servers |
| ScheduledRun | Scheduled agent executions |

---

## Controllers

10 controllers in `go/core/internal/controller/`:
- agent, sandboxagent, agentharness
- modelconfig, modelproviderconfig
- remote_mcp_server, mcp_server_tool
- scheduledrun (+ scheduler), service

---

## Language Guidelines

### When to Use Each Language

| Language | Use For | Don't Use For |
|----------|---------|---------------|
| **Go** | K8s controllers, CLI tools, core APIs, HTTP server, database layer | Agent runtime, LLM integrations, UI |
| **Python** | Agent runtime, ADK, LLM integrations, AI/ML logic | Kubernetes controllers, CLI, infrastructure |
| **TypeScript** | Web UI components and API clients only | Backend logic, controllers, agents |

**Rule of thumb:** Infrastructure in Go, AI/Agent logic in Python, User interface in TypeScript.

---

## Core Conventions

### Error Handling

**Go:**
```go
// Always wrap errors with context using %w
if err != nil {
    return fmt.Errorf("failed to create agent %s: %w", name, err)
}
```

**Controllers:**
```go
// Return error to requeue with backoff
if err != nil {
    return ctrl.Result{}, fmt.Errorf("reconciliation failed: %w", err)
}
```

### Testing

**Required for all PRs:**
- Unit tests for new functions/methods
- E2E tests for new CRD fields or API endpoints
- Mock external services (LLMs, K8s API) in unit tests
- All tests passing in CI pipeline

**Go testing pattern (table-driven):**
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {name: "valid input", input: "foo", want: "bar", wantErr: false},
        {name: "invalid input", input: "", want: "", wantErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Something(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Something() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("Something() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Commit Messages

Use **Conventional Commits** format:

```
<type>: <description>

[optional body]
```

**Types:** `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `perf`, `ci`

---

## API Versioning

- **v1alpha2** (current) - All new features go here
- **v1alpha1** (legacy/deprecated) - Minimal maintenance only

Breaking changes are acceptable in alpha versions.

---

## Best Practices

### Do's

- Read existing code before making changes
- Follow the language guidelines (Go for infra, Python for agents, TS for UI)
- Write table-driven tests in Go
- Wrap errors with context using `%w`
- Use conventional commit messages
- Mock external services in unit tests
- Update documentation for user-facing changes
- Run `make lint` before submitting

### Don'ts

- Don't add features beyond what's requested (avoid over-engineering)
- Don't modify v1alpha1 unless fixing critical bugs (focus on v1alpha2)
- Don't vendor dependencies (use go.mod)
- Don't commit without testing locally first
- Don't use `any` type in TypeScript
- Don't skip E2E tests for API/CRD changes
- Don't create new MCP servers in the main kagent repo

---

## Quick Reference

| Task | Command |
|------|---------|
| Create Kind cluster | `make create-kind-cluster` |
| Deploy kagent | `make helm-install` |
| Build all | `make build` |
| Build specific | `make build-controller`, `build-ui`, `build-app`, `build-golang-adk` |
| Run Go tests | `make -C go test` |
| Run E2E tests | `cd go && go test ./core/test/e2e/ -v -run TestE2E` |
| Lint code | `make -C go lint` |
| Generate CRD code | `make -C go generate` |
| Install CLI locally | `make kagent-cli-install` |
| Port-forward UI | `make kagent-ui-port-forward` |
| Push test agent | `make push-test-agent` |

---

## Additional Resources

- **Architecture:** See [docs/architecture/](docs/architecture/)
- **E2E Tests:** See [go/core/test/e2e/README.md](go/core/test/e2e/README.md)
- **Examples:** Check `python/samples/`
- **Helm:** See [helm/README.md](helm/README.md)

---

**Project Version:** v0.x.x (Alpha)
