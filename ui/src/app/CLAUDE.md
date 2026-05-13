# ui/src/app/ - App Router Routes

## Page Routes

| Route | Purpose |
|-------|---------|
| `agents/` | Agent listing, creation, and chat (dynamic: `[namespace]/[name]/chat/[chatId]`) |
| `a2a/` | A2A agent interaction (dynamic: `[namespace]/[agentName]`) |
| `a2a-sandboxes/` | Sandbox A2A agents (dynamic: `[namespace]/[agentName]`) |
| `models/` | Model config management (has `new/` creation page) |
| `mcp/` | MCP server management (has `new/` creation page) |
| `prompts/` | Prompt template management (dynamic: `[namespace]/[name]`) |
| `schedules/` | Scheduled run management (dynamic: `[namespace]/[name]`) |
| `dashboard/` | Platform dashboard |
| `login/` | Authentication page |
| `servers/` | Server listing |
| `tools/` | Tool listing |

## API Routes

| Route | Purpose |
|-------|---------|
| `api/config/` | Configuration endpoint |
| `api/stats/` | Platform statistics |
| `api/toolservers/` | Tool server proxy |

## Server Actions

| Route | Purpose |
|-------|---------|
| `actions/api/auth/` | Auth actions (GitHub OAuth callbacks, user info) |
| `actions/api/` | Additional server actions (agents, feedback, memories, models, sessions, tools, etc.) |
