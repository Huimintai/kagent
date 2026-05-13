# python/samples/ - Example Agents

Sample agent implementations demonstrating each supported framework.

| Directory | Framework | Examples |
|-----------|-----------|----------|
| `adk/` | Google ADK (kagent-adk) | Basic agent |
| `langgraph/` | LangGraph (kagent-langgraph) | Currency, HITL-tools, Kebab |
| `openai/` | OpenAI Agents (kagent-openai) | Basic agent |

Each sample is a standalone UV workspace member with its own `pyproject.toml`, `Dockerfile`, and `agent.yaml` (Kubernetes manifest).
