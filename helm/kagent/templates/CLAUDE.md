# helm/kagent/templates/ - Manifest Templates

## Subdirectories

| Directory | Content |
|-----------|---------|
| `rbac/` | RBAC resources (getter-role, writer-role, leader-election, bindings) |

## Template Files

| Template | Resource |
|----------|----------|
| `controller-deployment.yaml` | Controller Deployment |
| `controller-service.yaml` | Controller Service |
| `controller-serviceaccount.yaml` | Controller ServiceAccount |
| `controller-configmap.yaml` | Controller ConfigMap |
| `platform-configmap.yaml` | Platform ConfigMap |
| `ui-deployment.yaml` | UI Deployment |
| `ui-service.yaml` | UI Service |
| `ui-serviceaccount.yaml` | UI ServiceAccount |
| `ui-configmap.yaml` | UI ConfigMap |
| `ui-nginx-configmap.yaml` | UI Nginx config |
| `postgresql.yaml` | PostgreSQL StatefulSet |
| `postgresql-secret.yaml` | PostgreSQL credentials |
| `modelconfig.yaml` | Default ModelConfig |
| `modelconfig-secret.yaml` | Model API key secret |
| `toolserver-kagent.yaml` | Built-in ToolServer |
| `oauth2-proxy-templates.yaml` | OAuth2 Proxy (optional) |
| `openshift-route.yaml` | OpenShift Route (optional) |
| `builtin-prompts-configmap.yaml` | Built-in prompt templates |
| `_helpers.tpl` | Template helper functions |
| `NOTES.txt` | Post-install notes |
