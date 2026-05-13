# go/api/openshell/ - OpenShell API Types

Protocol Buffer definitions and generated Go code for the OpenShell sandboxing and inference service.

## Sub-packages

| Directory | Role |
|-----------|------|
| `proto/` | Protobuf source definitions (buf-managed) |
| `gen/` | Generated Go stubs (DO NOT EDIT) |

## Proto Files (proto/)

| File | Domain |
|------|--------|
| `openshell.proto` | Core OpenShell service |
| `sandbox.proto` | Sandbox lifecycle |
| `inference.proto` | LLM inference |
| `datamodel.proto` | Shared data models |
| `compute_driver.proto` | Compute driver interface |
| `test.proto` | Test service definitions |

## Workflow

1. Edit `.proto` files in `proto/`
2. Run `buf generate` (config in `buf.yaml` / `buf.gen.yaml`)
3. Generated Go code appears in `gen/`
