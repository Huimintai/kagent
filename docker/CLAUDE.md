# docker/ — Docker Build Contexts

Container image build resources for platform components.

## Sub-packages

| Directory | Role |
|-----------|------|
| `skills-init/` | Init container that pre-loads skills into agent pods |

## skills-init/

| File | Role |
|------|------|
| `Dockerfile` | Production multi-stage build |
| `Dockerfile.local` | Local development build |
| `go-containerregistry` | Binary for OCI image operations |
