"""Tests for SetMcpTokenTool and per-user session-token support in create_header_provider."""

from __future__ import annotations

from unittest.mock import AsyncMock, MagicMock

import pytest

from kagent.adk.tools.set_mcp_token_tool import MCP_TOKEN_STATE_PREFIX, SetMcpTokenTool, mcp_token_state_key
from kagent.adk.types import create_header_provider


class MockReadonlyContext:
    """Minimal ReadonlyContext mock that exposes a mutable state dict."""

    def __init__(self, state: dict | None = None):
        self._state: dict = state or {}

    @property
    def state(self):
        return self._state


# ---------------------------------------------------------------------------
# mcp_token_state_key helper
# ---------------------------------------------------------------------------


class TestMcpTokenStateKey:
    def test_format(self):
        assert mcp_token_state_key("github") == "mcp_token:github"

    def test_prefix_constant(self):
        assert MCP_TOKEN_STATE_PREFIX == "mcp_token"

    def test_different_labels(self):
        assert mcp_token_state_key("ado") != mcp_token_state_key("github")
        assert mcp_token_state_key("ado") == "mcp_token:ado"


# ---------------------------------------------------------------------------
# SetMcpTokenTool
# ---------------------------------------------------------------------------


class TestSetMcpTokenTool:
    @pytest.fixture
    def tool(self) -> SetMcpTokenTool:
        return SetMcpTokenTool()

    def _make_tool_context(self, state: dict | None = None) -> MagicMock:
        ctx = MagicMock()
        ctx.state = state if state is not None else {}
        return ctx

    @pytest.mark.asyncio
    async def test_stores_token_in_state(self, tool):
        state: dict = {}
        ctx = self._make_tool_context(state)
        result = await tool.run_async(args={"server_label": "github", "token": "ghp_abc123"}, tool_context=ctx)

        assert result["status"] == "ok"
        assert state["mcp_token:github"] == "ghp_abc123"

    @pytest.mark.asyncio
    async def test_stores_token_with_different_labels(self, tool):
        state: dict = {}
        ctx = self._make_tool_context(state)

        await tool.run_async(args={"server_label": "github", "token": "ghp_abc"}, tool_context=ctx)
        await tool.run_async(args={"server_label": "ado", "token": "ado_pat_xyz"}, tool_context=ctx)

        assert state["mcp_token:github"] == "ghp_abc"
        assert state["mcp_token:ado"] == "ado_pat_xyz"

    @pytest.mark.asyncio
    async def test_overwrites_existing_token(self, tool):
        state: dict = {"mcp_token:github": "old_token"}
        ctx = self._make_tool_context(state)
        await tool.run_async(args={"server_label": "github", "token": "new_token"}, tool_context=ctx)
        assert state["mcp_token:github"] == "new_token"

    @pytest.mark.asyncio
    async def test_strips_whitespace_from_token(self, tool):
        state: dict = {}
        ctx = self._make_tool_context(state)
        await tool.run_async(args={"server_label": "github", "token": "  ghp_xyz  "}, tool_context=ctx)
        assert state["mcp_token:github"] == "ghp_xyz"

    @pytest.mark.asyncio
    async def test_returns_error_when_server_label_missing(self, tool):
        ctx = self._make_tool_context()
        result = await tool.run_async(args={"server_label": "", "token": "ghp_abc"}, tool_context=ctx)
        assert result["status"] == "error"
        assert "server_label" in result["message"]

    @pytest.mark.asyncio
    async def test_returns_error_when_token_missing(self, tool):
        ctx = self._make_tool_context()
        result = await tool.run_async(args={"server_label": "github", "token": ""}, tool_context=ctx)
        assert result["status"] == "error"
        assert "token" in result["message"]

    def test_tool_name(self, tool):
        assert tool.name == "set_mcp_token"

    def test_declaration_has_required_fields(self, tool):
        decl = tool._get_declaration()
        assert decl.name == "set_mcp_token"
        assert "server_label" in decl.parameters.properties
        assert "token" in decl.parameters.properties
        assert "server_label" in decl.parameters.required
        assert "token" in decl.parameters.required


# ---------------------------------------------------------------------------
# create_header_provider with session_token_label
# ---------------------------------------------------------------------------


class TestCreateHeaderProviderSessionToken:
    def test_returns_none_when_no_args(self):
        assert create_header_provider() is None

    def test_returns_provider_when_session_token_label_set(self):
        provider = create_header_provider(session_token_label="github")
        assert provider is not None

    def test_injects_authorization_from_session_state(self):
        provider = create_header_provider(session_token_label="github")
        ctx = MockReadonlyContext(state={"mcp_token:github": "ghp_secret"})
        headers = provider(ctx)
        assert headers == {"Authorization": "Bearer ghp_secret"}

    def test_no_header_when_token_not_in_state(self):
        provider = create_header_provider(session_token_label="github")
        ctx = MockReadonlyContext(state={})
        headers = provider(ctx)
        assert headers == {}

    def test_no_header_when_context_is_none(self):
        provider = create_header_provider(session_token_label="github")
        headers = provider(None)
        assert headers == {}

    def test_session_token_used_when_no_sts(self):
        provider = create_header_provider(session_token_label="ado")
        ctx = MockReadonlyContext(state={"mcp_token:ado": "ado_pat_123"})
        headers = provider(ctx)
        assert headers["Authorization"] == "Bearer ado_pat_123"

    def test_sts_overrides_session_token(self):
        """STS tokens always win over the per-user session token."""

        def sts_provider(ctx):
            return {"Authorization": "Bearer sts-service-token"}

        provider = create_header_provider(
            session_token_label="github",
            sts_header_provider=sts_provider,
        )
        ctx = MockReadonlyContext(state={"mcp_token:github": "ghp_user_token"})
        headers = provider(ctx)
        # STS must win
        assert headers["Authorization"] == "Bearer sts-service-token"
        assert len(headers) == 1

    def test_allowed_headers_override_session_token_for_same_header(self):
        """Explicitly-allowed Authorization from A2A request overrides session token."""
        provider = create_header_provider(
            session_token_label="github",
            allowed_headers=["Authorization"],
        )
        ctx = MockReadonlyContext(
            state={
                "mcp_token:github": "ghp_session_token",
                "headers": {
                    "Authorization": "Bearer a2a-request-token",
                },
            }
        )
        headers = provider(ctx)
        assert headers["Authorization"] == "Bearer a2a-request-token"

    def test_session_token_does_not_interfere_with_other_labels(self):
        """Different label → token for 'ado' should not appear for 'github' provider."""
        provider = create_header_provider(session_token_label="github")
        ctx = MockReadonlyContext(state={"mcp_token:ado": "ado_token"})
        headers = provider(ctx)
        assert headers == {}

    def test_combined_session_token_and_allowed_headers(self):
        """Session token injects Authorization; other allowed headers are also included."""
        provider = create_header_provider(
            session_token_label="github",
            allowed_headers=["x-user-id"],
        )
        ctx = MockReadonlyContext(
            state={
                "mcp_token:github": "ghp_abc",
                "headers": {"x-user-id": "user-42"},
            }
        )
        headers = provider(ctx)
        assert headers["Authorization"] == "Bearer ghp_abc"
        assert headers["x-user-id"] == "user-42"
