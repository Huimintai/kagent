---
name: kagent-env-kind
description: >
  Configure secrets, ModelConfigs, and SAPAICore credentials for a local Kind cluster.
  Use this skill when setting up LLM provider access (SAPAICore, OpenAI, etc.) after
  kagent is deployed to Kind.
---

# Kagent Environment — Kind Cluster

Post-deployment configuration for secrets, ModelConfigs, and LLM provider access on a local Kind cluster.

**Prerequisite:** kagent must already be deployed (use `kagent-ci-kind` skill for build & deploy).

## SAPAICore Configuration

### Step 1: Create the credential secret

```bash
kubectl create secret generic sap-aicore-creds -n kagent \
  --from-literal=client_id='YOUR_CLIENT_ID' \
  --from-literal=client_secret='YOUR_CLIENT_SECRET' \
  --from-literal=auth_url='https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com' \
  --context kind-kagent \
  --dry-run=client -o yaml | kubectl apply --context kind-kagent -f -
```

### Step 2: Create ModelConfigs

```bash
cat <<'EOF' | kubectl apply --context kind-kagent -f -
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-claude-46-opus
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: anthropic--claude-4.6-opus
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-claude-46-sonnet
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: anthropic--claude-4.6-sonnet
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-gpt5
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: gpt-5
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
---
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: sap-aicore-text-embedding-3-small
  namespace: kagent
spec:
  apiKeySecret: sap-aicore-creds
  apiKeySecretKey: client_id
  model: text-embedding-3-small
  provider: SAPAICore
  sapAICore:
    authUrl: https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com
    baseUrl: https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com
    resourceGroup: default
EOF
```

### Step 3: Verify

```bash
kubectl get modelconfig -n kagent --context kind-kagent
# Should show sap-aicore-* entries with status Accepted
```

## SAPAICore Endpoint Reference

| Field | Value |
|-------|-------|
| Auth URL | `https://YOUR_TENANT.authentication.YOUR_REGION.hana.ondemand.com` |
| Base URL | `https://api.ai.YOUR_REGION.aws.ml.hana.ondemand.com` |
| Resource Group | `default` |
| Secret name | `sap-aicore-creds` (keys: `client_id`, `client_secret`, `auth_url`) |

## Notes

- SAPAICore is NOT a built-in Helm provider. The Helm chart installs with the default OpenAI provider.
- ModelConfigs are created separately after deployment via kubectl, not through Helm values.
- Actual credential values are stored in `.env` (gitignored) — never hardcode them in skill files.
