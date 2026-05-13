# ui/src/components/ - React Components

## Component Directories (with subdirectories)

| Directory | Role |
|-----------|------|
| `chat/` | Chat interface (messages, streaming, code blocks, tool calls, token stats) |
| `create/` | Agent creation form (model selection, tools, instructions, prompts) |
| `dashboard/` | Platform dashboard (stats, leaderboard, hot MCP servers) |
| `icons/` | Provider brand icons (Anthropic, Azure, Bedrock, Gemini, Ollama, OpenAI, SAP AI Core) |
| `mcp/` | MCP server form and listing |
| `models/` | Model config forms (has `new/` subdir with auth/params/basic sections) |
| `onboarding/` | Onboarding wizard (has `steps/` subdir with 6 step components) |
| `prompts/` | Prompt template editor (fragment entries) |
| `schedules/` | Scheduled run list and history table |
| `sidebars/` | Navigation sidebars (agent details, agent switcher, session groups, chat items) |
| `tools/` | Tool category filter |
| `ui/` | Radix-based primitives (button, dialog, form, input, select, tabs, etc.) |

## Standalone Components

| Component | Role |
|-----------|------|
| `AgentCard.tsx` | Agent card display |
| `AgentFilterToolbar.tsx` | Agent list filtering |
| `AgentGrid.tsx` / `AgentList.tsx` | Agent layout views |
| `AgentsProvider.tsx` | Agent data context provider |
| `AppInitializer.tsx` | App bootstrap logic |
| `Header.tsx` | App header with navigation |
| `Footer.tsx` | App footer |
| `ConfirmDialog.tsx` | Confirmation modal |
| `DeleteAgentButton.tsx` | Agent deletion action |
| `GitHubConnectButton.tsx` | GitHub OAuth connection |
| `SettingsModal.tsx` | User settings |
| `TokenExpiryBanner.tsx` | Auth token expiry warning |
| `ThemeProvider.tsx` / `ThemeToggle.tsx` | Dark/light theme |
| `UserMenu.tsx` | User dropdown menu |
| `ModelCombobox.tsx` / `ModelProviderCombobox.tsx` / `ProviderCombobox.tsx` | Selection widgets |
| `NamespaceCombobox.tsx` / `CategoryCombobox.tsx` | Filtering widgets |
| `AddServerDialog.tsx` | Server addition modal |
| `MemoriesDialog.tsx` | Memory viewer |
| `LoadingState.tsx` / `ErrorState.tsx` | Status displays |
| `Identicon.tsx` | Avatar generation |
