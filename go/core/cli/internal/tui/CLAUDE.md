# cli/internal/tui/

Terminal UI for interactive agent chat using Bubble Tea framework.

`workspace.go` implements the main TUI workspace model with agent selection and message streaming. `chat.go` handles the chat interface rendering.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `dialogs/` | Modal dialog components: agent chooser, MCP server wizard, dialog manager |
| `keys/` | Key bindings definition for the TUI |
| `theme/` | Color scheme and styling constants |
