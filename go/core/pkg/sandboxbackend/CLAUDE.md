# pkg/sandboxbackend/

Sandbox workload backend abstraction. Defines the `Backend` interface for building sandbox CRD objects and evaluating their readiness.

`backend.go` defines the interface. `filter_translator_owned.go` filters owned resource types based on whether an agent runs in sandbox mode.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `agentsxk8s/` | Implementation using kubernetes-sigs/agent-sandbox `Sandbox` CRD; builds Sandbox objects from pod templates and evaluates readiness from Sandbox status |
