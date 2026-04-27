---
name: kagent-test
description: >
  Post-deployment smoke test for kagent on Kind. Tests CRUD operations
  for agents, models, sessions, and tools via the HTTP API. Validates
  connectivity and new features. Use after rebase, feature work, or
  deployment changes.
---

# Kagent Test — Post-Deployment Smoke Test

Run this skill after deploying to a Kind cluster to verify nothing is broken.
Tests are organized in phases: health → model configs → agents → sessions → tools → feature flags → SRT sandbox.
Each CRUD phase creates test resources, verifies them, then cleans up.

## Prerequisites

- Kind cluster `kagent` running with controller + UI deployed
- Either port-forward active or MetalLB reachable

## Connectivity Setup

Detect the API and UI base URLs:

```bash
# Try port-forward first, fall back to MetalLB
if curl -sf http://localhost:8083/health >/dev/null 2>&1; then
  API_BASE="http://localhost:8083"
elif curl -sf http://172.18.255.1:8083/health >/dev/null 2>&1; then
  API_BASE="http://172.18.255.1:8083"
else
  echo "FAIL: Cannot reach controller API. Start port-forward:"
  echo "  kubectl port-forward -n kagent --context kind-kagent svc/kagent-controller 8083:8083 &"
  exit 1
fi

if curl -sf http://localhost:8082/health >/dev/null 2>&1; then
  UI_BASE="http://localhost:8082"
elif curl -sf http://172.18.255.0:8080/health >/dev/null 2>&1; then
  UI_BASE="http://172.18.255.0:8080"
else
  UI_BASE=""
  echo "WARN: Cannot reach UI. Some tests will be skipped."
fi

echo "API: $API_BASE"
echo "UI:  $UI_BASE"
```

Test namespace: use `default` (always exists). All test resource names prefixed with `kagent-test-`.

## Phase 1: Health & Cluster Check

```bash
# 1.1 Pods running
kubectl get pods -n kagent --context kind-kagent --no-headers | \
  awk '{if ($3 != "Running") print "WARN: " $1 " is " $3}'

# 1.2 Controller health
curl -sf $API_BASE/health && echo " OK: controller /health" || echo " FAIL: controller /health"

# 1.3 Auth mode
curl -sf $API_BASE/api/me | python3 -m json.tool
# In unsecure mode: returns default user. In trusted-proxy: returns claims from JWT.

# 1.4 Namespaces
curl -sf $API_BASE/api/namespaces | python3 -c "
import json,sys; d=json.load(sys.stdin)
ns = [n['name'] for n in d.get('data',[])]
print('OK: namespaces =', ns) if ns else print('FAIL: no namespaces')
"

# 1.5 Version
curl -sf $API_BASE/version | python3 -m json.tool
```

**Pass criteria:** Controller healthy, at least one namespace returned, version matches deployed commit.

## Phase 2: ModelConfig CRUD

```bash
NS="default"
NAME="kagent-test-model"
REF="$NS/$NAME"

# 2.1 CREATE
curl -sf -X POST $API_BASE/api/modelconfigs \
  -H 'Content-Type: application/json' \
  -d '{
    "ref": "'$REF'",
    "apiKey": "sk-test-dummy-key-000",
    "spec": {
      "model": "gpt-4o",
      "provider": "OpenAI",
      "openAI": {"baseUrl": "https://api.openai.com/v1"}
    }
  }' | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: created', d.get('data',{}).get('ref','?')) if not d.get('error') else print('FAIL: create -', d)
"

# 2.2 LIST & verify present
curl -sf $API_BASE/api/modelconfigs | python3 -c "
import json,sys; d=json.load(sys.stdin)
refs = [m['ref'] for m in d.get('data',[])]
print('OK: model in list') if '$REF' in refs else print('FAIL: model not in list, got:', refs)
"

# 2.3 GET by name
curl -sf $API_BASE/api/modelconfigs/$NS/$NAME | python3 -c "
import json,sys; d=json.load(sys.stdin)
m = d.get('data',{})
print('OK: get model, provider =', m.get('spec',{}).get('provider','?')) if not d.get('error') else print('FAIL:', d)
"

# 2.4 UPDATE
curl -sf -X PUT $API_BASE/api/modelconfigs/$NS/$NAME \
  -H 'Content-Type: application/json' \
  -d '{
    "spec": {
      "model": "gpt-4o-mini",
      "provider": "OpenAI",
      "openAI": {"baseUrl": "https://api.openai.com/v1"}
    }
  }' | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: updated model') if not d.get('error') else print('FAIL: update -', d)
"

# 2.5 DELETE
curl -sf -X DELETE $API_BASE/api/modelconfigs/$NS/$NAME | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: deleted model') if not d.get('error') else print('FAIL: delete -', d)
"
```

**Pass criteria:** All 5 operations succeed (create, list, get, update, delete).

## Phase 3: Agent CRUD

Requires a ModelConfig to exist first. Create one, then test agent CRUD.

```bash
NS="default"
MC_NAME="kagent-test-model-for-agent"
MC_REF="$NS/$MC_NAME"
AGENT_NAME="kagent-test-agent"

# 3.0 Setup: create model config
curl -sf -X POST $API_BASE/api/modelconfigs \
  -H 'Content-Type: application/json' \
  -d '{
    "ref": "'$MC_REF'",
    "apiKey": "sk-test-dummy-key-000",
    "spec": {"model": "gpt-4o", "provider": "OpenAI"}
  }' >/dev/null

# 3.1 CREATE agent (declarative with inline skill)
curl -sf -X POST $API_BASE/api/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "kagent.dev/v1alpha2",
    "kind": "Agent",
    "metadata": {"name": "'$AGENT_NAME'", "namespace": "'$NS'"},
    "spec": {
      "type": "Declarative",
      "description": "Smoke test agent",
      "declarative": {
        "modelConfig": "'$MC_NAME'",
        "systemMessage": "You are a test agent.",
        "tools": [],
        "inlineSkills": [
          {
            "name": "test-skill",
            "description": "A test inline skill",
            "content": "You can answer questions about testing."
          }
        ]
      }
    }
  }' | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: created agent') if not d.get('error') else print('FAIL: create agent -', d)
"

# 3.2 LIST & verify
curl -sf $API_BASE/api/agents | python3 -c "
import json,sys; d=json.load(sys.stdin)
names = [a['agent']['metadata']['name'] for a in d.get('data',[])]
print('OK: agent in list') if '$AGENT_NAME' in names else print('FAIL: agent not in list')
"

# 3.3 GET
curl -sf $API_BASE/api/agents/$NS/$AGENT_NAME | python3 -c "
import json,sys; d=json.load(sys.stdin)
a = d.get('data',{})
desc = a.get('agent',{}).get('spec',{}).get('description','')
print('OK: get agent, desc =', desc) if not d.get('error') else print('FAIL:', d)
"

# 3.4 DELETE agent
curl -sf -X DELETE $API_BASE/api/agents/$NS/$AGENT_NAME | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: deleted agent') if not d.get('error') else print('FAIL: delete agent -', d)
"

# 3.5 Cleanup model config
curl -sf -X DELETE $API_BASE/api/modelconfigs/$NS/$MC_NAME >/dev/null
```

**Pass criteria:** Agent created with inline skill, listed, retrieved, deleted.

## Phase 4: Session CRUD

Requires an agent. Create one, test sessions, then clean up.

```bash
NS="default"
MC_NAME="kagent-test-model-session"
AGENT_NAME="kagent-test-agent-session"

# 4.0 Setup
curl -sf -X POST $API_BASE/api/modelconfigs \
  -H 'Content-Type: application/json' \
  -d '{"ref":"'$NS/$MC_NAME'","apiKey":"sk-test","spec":{"model":"gpt-4o","provider":"OpenAI"}}' >/dev/null

curl -sf -X POST $API_BASE/api/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion":"kagent.dev/v1alpha2","kind":"Agent",
    "metadata":{"name":"'$AGENT_NAME'","namespace":"'$NS'"},
    "spec":{"type":"Declarative","description":"Session test agent",
      "declarative":{"modelConfig":"'$MC_NAME'","systemMessage":"Test.","tools":[]}}
  }' >/dev/null

# Wait for agent to be compiled
sleep 5

# 4.1 CREATE session
SESSION_ID=$(curl -sf -X POST $API_BASE/api/sessions \
  -H 'Content-Type: application/json' \
  -d '{"agent_ref":"'$NS/$AGENT_NAME'","name":"kagent-test-session"}' | \
  python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('data',{}).get('id',''))")
echo "Session ID: $SESSION_ID"
[ -n "$SESSION_ID" ] && echo "OK: created session" || echo "FAIL: no session ID"

# 4.2 LIST sessions
curl -sf $API_BASE/api/sessions | python3 -c "
import json,sys; d=json.load(sys.stdin)
ids = [s['id'] for s in d.get('data',[])]
print('OK: session in list') if '$SESSION_ID' in ids else print('WARN: session not in list (may be user-scoped)')
"

# 4.3 GET session
curl -sf $API_BASE/api/sessions/$SESSION_ID | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: get session') if not d.get('error') else print('FAIL: get session -', d)
"

# 4.4 DELETE session
curl -sf -X DELETE $API_BASE/api/sessions/$SESSION_ID | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: deleted session') if not d.get('error') else print('FAIL: delete session -', d)
"

# 4.5 Cleanup
curl -sf -X DELETE $API_BASE/api/agents/$NS/$AGENT_NAME >/dev/null
curl -sf -X DELETE $API_BASE/api/modelconfigs/$NS/$MC_NAME >/dev/null
```

**Pass criteria:** Session created, listed, retrieved, deleted.

## Phase 5: Tools, Models, & Providers

These are read-only checks on existing data.

```bash
# 5.1 Tool servers
curl -sf $API_BASE/api/toolservers | python3 -c "
import json,sys; d=json.load(sys.stdin)
servers = d.get('data',[])
print('OK: tool servers =', len(servers))
for s in servers[:5]:
    tools = [t['name'] for t in s.get('discoveredTools',[])]
    print('  ', s['ref'], ':', len(tools), 'tools')
"

# 5.2 Tools
curl -sf $API_BASE/api/tools | python3 -c "
import json,sys; d=json.load(sys.stdin)
tools = d.get('data',[])
print('OK: total tools =', len(tools))
"

# 5.3 Models — check SAP AI Core present
curl -sf $API_BASE/api/models | python3 -c "
import json,sys; d=json.load(sys.stdin)
providers = list(d.get('data',{}).keys())
print('OK: model providers =', providers)
has_sap = 'sap_ai_core' in providers
print('OK: SAP AI Core models present') if has_sap else print('WARN: SAP AI Core not in model list')
"

# 5.4 Provider metadata
curl -sf $API_BASE/api/modelproviderconfigs/models | python3 -c "
import json,sys; d=json.load(sys.stdin)
providers = d.get('data',[])
names = [p['name'] for p in providers]
print('OK: providers =', names)
print('OK: SAPAICore provider configured') if 'SAPAICore' in names else print('WARN: SAPAICore missing')
"

# 5.5 Configured providers (ModelProviderConfig CRDs)
curl -sf $API_BASE/api/modelproviderconfigs/configured | python3 -c "
import json,sys; d=json.load(sys.stdin)
items = d.get('data',[])
print('OK: configured providers =', len(items))
for p in items: print('  ', p.get('name'), '-', p.get('type'))
"
```

**Pass criteria:** Tool servers listed, SAP AI Core in model list, providers enumerated.

## Phase 6: Feature Flags & UI Config

```bash
# 6.1 UI feature flag endpoint (served by Next.js via nginx)
if [ -n "$UI_BASE" ]; then
  curl -sf $UI_BASE/api/config | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: UI config endpoint responds')
for k,v in d.items(): print('  ', k, '=', v)
" || echo "FAIL: /api/config not reachable"
else
  echo "SKIP: UI not reachable"
fi

# 6.2 ConfigMap values
kubectl get configmap kagent-ui-config -n kagent --context kind-kagent -o json 2>/dev/null | python3 -c "
import json,sys
try:
  cm = json.load(sys.stdin)
  data = cm.get('data',{})
  print('OK: kagent-ui-config ConfigMap')
  for k,v in sorted(data.items()): print('  ', k, '=', v)
except: print('WARN: kagent-ui-config ConfigMap not found')
"
```

## Phase 7: SRT Network Allowlist Auto-Discovery

Creates an agent with `executeCodeBlocks: true` in the `kagent` namespace (which has
multiple Services), then verifies the generated Secret's `srt-settings.json` contains:
- Auto-discovered same-namespace service DNS names in `network.allowedDomains`
- `enableWeakerNestedSandbox: true`
- `allowAllUnixSockets: true`

```bash
NS="kagent"
MC_NAME="kagent-test-model-srt"
MC_REF="$NS/$MC_NAME"
AGENT_NAME="kagent-test-agent-srt"

# 7.0 Setup: create model config
curl -sf -X POST $API_BASE/api/modelconfigs \
  -H 'Content-Type: application/json' \
  -d '{
    "ref": "'$MC_REF'",
    "apiKey": "sk-test-dummy-key-srt",
    "spec": {"model": "gpt-4o", "provider": "OpenAI"}
  }' >/dev/null

# 7.1 CREATE agent with executeCodeBlocks: true
curl -sf -X POST $API_BASE/api/agents \
  -H 'Content-Type: application/json' \
  -d '{
    "apiVersion": "kagent.dev/v1alpha2",
    "kind": "Agent",
    "metadata": {"name": "'$AGENT_NAME'", "namespace": "'$NS'"},
    "spec": {
      "type": "Declarative",
      "description": "SRT network allowlist smoke test",
      "declarative": {
        "modelConfig": "'$MC_NAME'",
        "systemMessage": "You are a test agent with code execution.",
        "executeCodeBlocks": true,
        "tools": []
      }
    }
  }' | python3 -c "
import json,sys; d=json.load(sys.stdin)
print('OK: created SRT test agent') if not d.get('error') else print('FAIL: create SRT agent -', d)
"

# 7.2 Wait for reconciliation
echo "Waiting for controller reconciliation..."
sleep 8

# 7.3 Verify srt-settings.json in the generated Secret
kubectl get secret $AGENT_NAME -n $NS --context kind-kagent -o jsonpath='{.data.srt-settings\.json}' 2>/dev/null | \
  base64 -d | python3 -c "
import json, sys

try:
    srt = json.loads(sys.stdin.read())
except:
    print('FAIL: srt-settings.json not found or invalid in Secret')
    sys.exit(1)

errors = []

# Check enableWeakerNestedSandbox
if srt.get('enableWeakerNestedSandbox') is not True:
    errors.append('enableWeakerNestedSandbox missing or not true')

# Check allowAllUnixSockets
if srt.get('allowAllUnixSockets') is not True:
    errors.append('allowAllUnixSockets missing or not true')

# Check allowedDomains contains auto-discovered services
domains = srt.get('network', {}).get('allowedDomains', [])
# kagent namespace should have services — check at least one <svc>.kagent entry
kagent_domains = [d for d in domains if d.endswith('.kagent')]
if not kagent_domains:
    errors.append(f'no auto-discovered .kagent domains in allowedDomains (got: {domains})')

if errors:
    for e in errors:
        print(f'FAIL: {e}')
    print('Full srt-settings.json:', json.dumps(srt, indent=2))
else:
    print(f'OK: enableWeakerNestedSandbox=true')
    print(f'OK: allowAllUnixSockets=true')
    print(f'OK: auto-discovered {len(kagent_domains)} service domains: {kagent_domains}')
"

# 7.4 Cleanup
curl -sf -X DELETE $API_BASE/api/agents/$NS/$AGENT_NAME >/dev/null
curl -sf -X DELETE $API_BASE/api/modelconfigs/$NS/$MC_NAME >/dev/null
echo "OK: SRT test resources cleaned up"
```

**Pass criteria:** Secret contains `enableWeakerNestedSandbox: true`, `allowAllUnixSockets: true`, and `allowedDomains` includes auto-discovered `<svc>.kagent` entries.

## Phase 8: Manual Checklist

After the automated tests pass, present this checklist to the user. These require a browser and human judgment.

**Ask the user to verify:**

```
Post-Deployment Manual Checklist
================================
Open the UI at $UI_BASE (or localhost:8082) and verify:

[ ] UI homepage loads without errors
[ ] Agent list page renders all agents
[ ] Agent detail sidebar opens correctly
[ ] Create new agent form works (select model, add tools)
[ ] Model list page renders
[ ] Model create page respects "disableModelCreation" flag (if enabled)
[ ] Tool servers page lists discovered tools
[ ] Start a chat session with an agent — messages render
[ ] WebSocket streaming works (responses appear incrementally)

If OIDC / GitHub OAuth is configured:
[ ] User identity displayed in header
[ ] GitHub Connect button visible and functional
[ ] OAuth flow completes without errors
[ ] Per-user agent filtering works (private_mode agents hidden from others)

If SAP AI Core is configured:
[ ] SAP AI Core model config can be created via UI
[ ] Chat with SAP AI Core model returns responses
```

## Execution Protocol

1. Run phases 1-7 sequentially. Each phase prints `OK:` or `FAIL:` per test.
2. If Phase 1 fails (health check), stop and fix deployment first.
3. Phases 2-4 create and delete their own test resources — no manual cleanup needed.
4. Phase 7 (SRT) creates resources in kagent namespace and cleans up automatically.
5. After Phase 7, present the Phase 8 checklist to the user.
6. Summarize results:

```
=== Smoke Test Summary ===
Phase 1 (Health):      X/5 passed
Phase 2 (ModelConfig): X/5 passed
Phase 3 (Agent CRUD):  X/5 passed
Phase 4 (Sessions):    X/4 passed
Phase 5 (Tools):       X/5 passed
Phase 6 (Flags):       X/2 passed
Phase 7 (SRT):         X/3 passed
Phase 8 (Manual):      Checklist presented to user
```

## Cleanup

If tests are interrupted, clean up stale resources:

```bash
NS="default"
kubectl delete agent kagent-test-agent kagent-test-agent-session -n $NS --context kind-kagent 2>/dev/null
kubectl delete modelconfig kagent-test-model kagent-test-model-for-agent kagent-test-model-session -n $NS --context kind-kagent 2>/dev/null
kubectl delete agent kagent-test-agent-srt -n kagent --context kind-kagent 2>/dev/null
kubectl delete modelconfig kagent-test-model-srt -n kagent --context kind-kagent 2>/dev/null
echo "Cleanup complete"
```
