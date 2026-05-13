# kagent-core - Shared Library

Common utilities shared across all kagent Python packages: configuration, A2A protocol, and OpenTelemetry tracing.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `kagent.core`) |
| `tests/` | Unit tests |

## Key Features

- A2A protocol utilities (config, constants, HITL, task store, task result aggregation)
- OpenTelemetry tracing (span processor, utilities)
- Configuration loading (`_config.py`)
- Logging setup (`_logging.py`)

## Dependencies

- `a2a-sdk[http-server]` >=0.3.23
- `opentelemetry-api`, `opentelemetry-sdk`, `opentelemetry-exporter-otlp-proto-grpc` (1.38.x)
- `opentelemetry-instrumentation-*` (OpenAI, Anthropic, httpx, FastAPI, Google GenAI)
