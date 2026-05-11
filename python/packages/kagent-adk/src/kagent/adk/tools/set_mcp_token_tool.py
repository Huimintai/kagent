"""Tool that lets the agent store a per-user MCP token into session state.

The token is stored under a well-known key in the ADK session state so that
the MCP header provider can pick it up dynamically on every MCP session
creation, enabling per-user authentication without sharing a static token.
"""

from __future__ import annotations

import logging
from typing import Any

from google.adk.tools.base_tool import BaseTool
from google.adk.tools.tool_context import ToolContext
from google.genai import types

logger = logging.getLogger(__name__)

# Well-known session state key for per-user MCP tokens.
# Key format: "mcp_token:<server_label>" — allows different tokens per MCP server.
MCP_TOKEN_STATE_PREFIX = "mcp_token"


def mcp_token_state_key(server_label: str) -> str:
    """Return the session state key for a given MCP server label."""
    return f"{MCP_TOKEN_STATE_PREFIX}:{server_label}"


class SetMcpTokenTool(BaseTool):
    """Built-in tool that stores a user-supplied token for a named MCP server
    into the current session state.

    The token is scoped to the current session only — it is never persisted to
    any external store.  When the session ends the token is gone.

    Usage by the agent:
        set_mcp_token(server_label="github", token="ghp_xxxxxxxxxxxx")

    The header provider in KAgentMcpToolset will pick up the token automatically
    on the next tool call.
    """

    def __init__(self) -> None:
        super().__init__(
            name="set_mcp_token",
            description=(
                "Store a personal access token or OAuth token for a specific MCP server "
                "so that subsequent tool calls are authenticated as the current user. "
                "The token is held in memory for this session only and is never stored permanently. "
                "Call this when a user provides their own token for a service like GitHub or Azure DevOps."
            ),
        )

    def _get_declaration(self) -> types.FunctionDeclaration:
        return types.FunctionDeclaration(
            name=self.name,
            description=self.description,
            parameters=types.Schema(
                type=types.Type.OBJECT,
                properties={
                    "server_label": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "Identifier for the MCP server, e.g. 'github' or 'ado'. "
                            "Must match the label configured in the agent's MCP tool list."
                        ),
                    ),
                    "token": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "The personal access token or OAuth bearer token. "
                            "For GitHub: starts with 'ghp_', 'gho_', or 'github_pat_'. "
                            "For Azure DevOps: a PAT from dev.azure.com."
                        ),
                    ),
                },
                required=["server_label", "token"],
            ),
        )

    async def run_async(
        self,
        *,
        args: dict[str, Any],
        tool_context: ToolContext,
    ) -> Any:
        server_label: str = args.get("server_label", "").strip()
        token: str = args.get("token", "").strip()

        if not server_label:
            return {"status": "error", "message": "server_label is required"}
        if not token:
            return {"status": "error", "message": "token is required"}

        state_key = mcp_token_state_key(server_label)
        tool_context.state[state_key] = token
        logger.info(
            "set_mcp_token: stored token for server '%s' (length=%d)",
            server_label,
            len(token),
        )
        return {
            "status": "ok",
            "message": f"Token for '{server_label}' stored for this session. Your subsequent requests will use it.",
        }
