# examples/

Example programs demonstrating Go ADK usage patterns.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `oneshot/` | Runs a single prompt against an agent config file and prints the response; demonstrates config-driven agent creation and both streaming and non-streaming modes |
| `byo/` | BYO (Bring Your Own) agent example; builds an AgentConfig programmatically with parallel sub-agents and exposes as an A2A-compatible server |
