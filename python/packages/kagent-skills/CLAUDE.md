# kagent-skills - Skill Discovery and Loading

Core library for discovering and loading agent skills from YAML definitions.

## Structure

| Directory | Purpose |
|-----------|---------|
| `src/` | Source code (namespace: `kagent.skills` + `kagent.tests`) |

## Key Modules (in `kagent.skills`)

- `discovery.py` - Skill file discovery
- `models.py` - Skill data models
- `prompts.py` - Prompt template handling
- `session.py` - Session-scoped skill state
- `shell.py` - Shell command skill execution

## Dependencies

- `pydantic` >=2.0.0
- `pyyaml` >=6.0
