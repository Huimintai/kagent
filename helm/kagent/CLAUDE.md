# helm/kagent/ - Main Application Chart

Deploys the complete kagent platform: controller, UI, PostgreSQL, RBAC, and optional OAuth2 proxy.

## Structure

| Directory | Purpose |
|-----------|---------|
| `templates/` | Kubernetes manifest templates |
| `files/` | Static config files (nginx.conf, supervisord.conf) |
| `tests/` | Helm unit tests (helm-unittest) |

## Key Files

- `values.yaml` - Default chart values
- `Chart-template.yaml` - Chart metadata template

## Tests

| Test | Validates |
|------|-----------|
| `controller-deployment_test.yaml` | Controller deployment spec |
| `controller-service_test.yaml` | Controller service |
| `modelconfig_test.yaml` | ModelConfig resource |
| `modelconfig-secret_test.yaml` | ModelConfig secret |
| `postgresql_test.yaml` | PostgreSQL deployment |
| `rbac_test.yaml` | RBAC rules |
| `security-context_test.yaml` | Security contexts |
| `toolserver_test.yaml` | ToolServer resource |
| `ui-deployment_test.yaml` | UI deployment |
| `ui-nginx-configmap_test.yaml` | UI nginx config |
| `ui-service_test.yaml` | UI service |
