"""Tools for MCP OAuth 2.1 Authorization Code Flow with PKCE.

These tools enable agents to authenticate with OAuth-protected MCP servers
(e.g. github-mcp-server on github.tools.sap) without requiring a pre-shared token.

Flow:
  1. Agent calls initiate_mcp_oauth(server_label, server_url) when encountering
     a 401 from an MCP server.
  2. Tool discovers OAuth metadata (PRM + OASM), generates PKCE params, and builds
     the authorization URL.
  3. Agent presents the URL to the user (via ask_user) and waits for them to paste
     back the authorization code.
  4. Agent calls complete_mcp_oauth(server_label, code, state) to exchange the code
     for an access token.
  5. Token is stored in session state under mcp_token:<server_label>, which the
     existing create_header_provider() picks up automatically.
"""

from __future__ import annotations

import hashlib
import base64
import logging
import secrets
import string
from typing import Any
from urllib.parse import urlencode, urljoin, urlparse

import httpx
from google.adk.tools.base_tool import BaseTool
from google.adk.tools.tool_context import ToolContext
from google.genai import types

from kagent.adk.tools.set_mcp_token_tool import mcp_token_state_key

logger = logging.getLogger(__name__)

# Session state key prefix for PKCE verifier (transient, used only during OAuth flow).
_PKCE_STATE_PREFIX = "_mcp_oauth_pkce"
_OAUTH_STATE_PREFIX = "_mcp_oauth_state"


def _pkce_state_key(server_label: str) -> str:
    return f"{_PKCE_STATE_PREFIX}:{server_label}"


def _oauth_state_key(server_label: str) -> str:
    return f"{_OAUTH_STATE_PREFIX}:{server_label}"


def _generate_pkce() -> tuple[str, str, str]:
    """Generate PKCE verifier, challenge, and OAuth state."""
    verifier = "".join(secrets.choice(string.ascii_letters + string.digits + "-._~") for _ in range(128))
    digest = hashlib.sha256(verifier.encode()).digest()
    challenge = base64.urlsafe_b64encode(digest).decode().rstrip("=")
    state = secrets.token_urlsafe(32)
    return verifier, challenge, state


async def _discover_oauth_metadata(server_url: str) -> dict[str, Any]:
    """Discover OAuth metadata for an MCP server.

    Follows the MCP OAuth 2.1 spec:
    1. GET /.well-known/oauth-protected-resource/mcp → PRM → auth_server_url
    2. GET <auth_server>/.well-known/oauth-authorization-server → OASM

    Returns a dict with keys: auth_endpoint, token_endpoint, registration_endpoint,
    authorization_server (used for client registration).
    """
    parsed = urlparse(server_url)
    base_url = f"{parsed.scheme}://{parsed.netloc}"

    async with httpx.AsyncClient(timeout=10.0, verify=False) as client:
        # Step 1: Discover Protected Resource Metadata
        prm_url = urljoin(base_url, "/.well-known/oauth-protected-resource/mcp")
        try:
            prm_resp = await client.get(prm_url)
            if prm_resp.status_code == 200:
                prm = prm_resp.json()
                auth_servers = prm.get("authorization_servers", [])
                auth_server_url = auth_servers[0] if auth_servers else base_url
            else:
                auth_server_url = base_url
        except Exception:
            auth_server_url = base_url

        # Step 2: Discover OAuth Authorization Server Metadata
        parsed_auth = urlparse(auth_server_url)
        auth_base = f"{parsed_auth.scheme}://{parsed_auth.netloc}"
        oasm_url = urljoin(auth_base, "/.well-known/oauth-authorization-server")
        try:
            oasm_resp = await client.get(oasm_url)
            if oasm_resp.status_code == 200:
                oasm = oasm_resp.json()
                return {
                    "auth_endpoint": oasm.get("authorization_endpoint", urljoin(auth_base, "/login/oauth/authorize")),
                    "token_endpoint": oasm.get("token_endpoint", urljoin(auth_base, "/login/oauth/access_token")),
                    "registration_endpoint": oasm.get("registration_endpoint"),
                    "auth_server_url": auth_server_url,
                    "scopes_supported": oasm.get("scopes_supported", []),
                }
        except Exception:
            pass

        # Fallback: use GitHub Enterprise default paths
        return {
            "auth_endpoint": urljoin(auth_base, "/login/oauth/authorize"),
            "token_endpoint": urljoin(auth_base, "/login/oauth/access_token"),
            "registration_endpoint": None,
            "auth_server_url": auth_server_url,
            "scopes_supported": [],
        }


async def _register_oauth_client(
    registration_endpoint: str,
    client_name: str,
    redirect_uri: str,
    scopes: list[str],
) -> dict[str, Any] | None:
    """Attempt dynamic client registration (RFC 7591).

    Returns client_id and client_secret if successful, None otherwise.
    """
    try:
        async with httpx.AsyncClient(timeout=10.0, verify=False) as client:
            resp = await client.post(
                registration_endpoint,
                json={
                    "client_name": client_name,
                    "redirect_uris": [redirect_uri],
                    "grant_types": ["authorization_code"],
                    "response_types": ["code"],
                    "token_endpoint_auth_method": "none",
                    "scope": " ".join(scopes),
                },
            )
            if resp.status_code in (200, 201):
                return resp.json()
    except Exception as e:
        logger.debug("Dynamic client registration failed: %s", e)
    return None


class InitiateMcpOAuthTool(BaseTool):
    """Tool that starts the OAuth 2.1 Authorization Code + PKCE flow for an MCP server.

    The tool discovers OAuth endpoints, generates PKCE parameters, and builds the
    authorization URL. It returns the URL for the agent to present to the user.

    The user must open the URL, authorize the app, and then provide either:
    - The full redirect URL (containing ?code=xxx&state=yyy), or
    - Just the authorization code.

    Then the agent must call complete_mcp_oauth() to finish the token exchange.
    """

    def __init__(self) -> None:
        super().__init__(
            name="initiate_mcp_oauth",
            description=(
                "Start an OAuth 2.1 authorization flow to authenticate with an MCP server. "
                "Call this when the MCP server returns a 401 Unauthorized error and no token has been set. "
                "The tool will return an authorization URL that the user must open in their browser. "
                "After authorizing, the user pastes back the authorization code or the full redirect URL. "
                "Then call complete_mcp_oauth() to finish authentication."
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
                        description="MCP server label, e.g. 'github'. Must match the sessionTokenLabel in the agent config.",
                    ),
                    "server_url": types.Schema(
                        type=types.Type.STRING,
                        description="Full URL of the MCP server, e.g. 'https://github-mcp.tools.sap/mcp'.",
                    ),
                    "client_id": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "OAuth client ID for the application. Required if the server does not support "
                            "dynamic client registration. Leave empty to attempt dynamic registration."
                        ),
                    ),
                    "redirect_uri": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "OAuth redirect URI. Use 'urn:ietf:wg:oauth:2.0:oob' for manual code pasting "
                            "(recommended for agent flows). Default: 'urn:ietf:wg:oauth:2.0:oob'."
                        ),
                    ),
                    "scope": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "Space-separated OAuth scopes to request. "
                            "For GitHub: 'repo read:org read:user'. Leave empty to use server defaults."
                        ),
                    ),
                },
                required=["server_label", "server_url"],
            ),
        )

    async def run_async(self, *, args: dict[str, Any], tool_context: ToolContext) -> Any:
        server_label = args.get("server_label", "").strip()
        server_url = args.get("server_url", "").strip()
        client_id = args.get("client_id", "").strip()
        redirect_uri = args.get("redirect_uri", "").strip() or "urn:ietf:wg:oauth:2.0:oob"
        scope = args.get("scope", "").strip()

        if not server_label:
            return {"status": "error", "message": "server_label is required"}
        if not server_url:
            return {"status": "error", "message": "server_url is required"}

        try:
            # Discover OAuth endpoints
            metadata = await _discover_oauth_metadata(server_url)
            auth_endpoint = metadata["auth_endpoint"]
            registration_endpoint = metadata.get("registration_endpoint")
            scopes_supported = metadata.get("scopes_supported", [])

            # Determine scope
            if not scope and scopes_supported:
                scope = " ".join(scopes_supported[:5])  # use first 5 scopes as default

            # Attempt dynamic registration if no client_id provided
            if not client_id and registration_endpoint:
                reg_result = await _register_oauth_client(
                    registration_endpoint=registration_endpoint,
                    client_name="kagent",
                    redirect_uri=redirect_uri,
                    scopes=scope.split() if scope else [],
                )
                if reg_result and reg_result.get("client_id"):
                    client_id = reg_result["client_id"]
                    logger.info("Dynamic client registration succeeded: client_id=%s", client_id)

            if not client_id:
                return {
                    "status": "error",
                    "message": (
                        "No OAuth client_id available. "
                        "Please provide a client_id or ensure the server supports dynamic client registration. "
                        "To get a client_id: go to github.tools.sap → Settings → Developer settings → "
                        "OAuth Apps → New OAuth App, then provide the client_id here."
                    ),
                }

            # Generate PKCE parameters
            verifier, challenge, state = _generate_pkce()

            # Build authorization URL
            auth_params: dict[str, str] = {
                "response_type": "code",
                "client_id": client_id,
                "redirect_uri": redirect_uri,
                "state": state,
                "code_challenge": challenge,
                "code_challenge_method": "S256",
            }
            if scope:
                auth_params["scope"] = scope

            authorization_url = f"{auth_endpoint}?{urlencode(auth_params)}"

            # Store PKCE verifier, state, client_id, and token endpoint in session state for complete step
            tool_context.state[_pkce_state_key(server_label)] = verifier
            tool_context.state[_oauth_state_key(server_label)] = state
            tool_context.state[f"_mcp_oauth_client_id:{server_label}"] = client_id
            tool_context.state[f"_mcp_oauth_token_endpoint:{server_label}"] = metadata["token_endpoint"]
            tool_context.state[f"_mcp_oauth_redirect_uri:{server_label}"] = redirect_uri

            logger.info(
                "initiate_mcp_oauth: OAuth flow started for server '%s', state=%s",
                server_label,
                state[:8] + "...",
            )

            return {
                "status": "ok",
                "authorization_url": authorization_url,
                "instructions": (
                    f"Please open the following URL in your browser to authorize access:\n\n"
                    f"{authorization_url}\n\n"
                    f"After authorizing, GitHub will redirect you to a page showing an authorization code "
                    f"(or redirect to '{redirect_uri}'). "
                    f"Please copy the authorization code (the value after 'code=') and provide it to me. "
                    f"Then I will call complete_mcp_oauth() to finish the authentication."
                ),
            }

        except Exception as e:
            logger.exception("initiate_mcp_oauth failed for server '%s'", server_label)
            return {"status": "error", "message": f"OAuth initiation failed: {e}"}


class CompleteMcpOAuthTool(BaseTool):
    """Tool that completes the OAuth 2.1 token exchange after the user has authorized.

    Receives the authorization code (from the user), uses the stored PKCE verifier
    to exchange it for an access token, and stores the token in session state.

    After this tool succeeds, subsequent MCP tool calls will automatically use the
    OAuth access token via the session token mechanism.
    """

    def __init__(self) -> None:
        super().__init__(
            name="complete_mcp_oauth",
            description=(
                "Complete an OAuth 2.1 token exchange after the user has authorized access. "
                "Call this after initiate_mcp_oauth() once the user provides the authorization code. "
                "The authorization code is the 'code' parameter from the redirect URL after GitHub authorization. "
                "On success, the MCP server will be authenticated for this session."
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
                        description="MCP server label, must match what was used in initiate_mcp_oauth().",
                    ),
                    "code": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "The authorization code from the OAuth callback. "
                            "This is the 'code=' parameter from the redirect URL, "
                            "or the code shown on the GitHub authorization page."
                        ),
                    ),
                    "state": types.Schema(
                        type=types.Type.STRING,
                        description=(
                            "Optional: the 'state' parameter from the redirect URL. "
                            "Used to verify the flow integrity. "
                            "If the user pasted the full redirect URL, extract the state value. "
                            "Can be omitted if not available."
                        ),
                    ),
                },
                required=["server_label", "code"],
            ),
        )

    async def run_async(self, *, args: dict[str, Any], tool_context: ToolContext) -> Any:
        server_label = args.get("server_label", "").strip()
        code = args.get("code", "").strip()
        provided_state = args.get("state", "").strip()

        if not server_label:
            return {"status": "error", "message": "server_label is required"}
        if not code:
            return {"status": "error", "message": "code is required"}

        # Retrieve stored OAuth flow parameters
        verifier = tool_context.state.get(_pkce_state_key(server_label))
        expected_state = tool_context.state.get(_oauth_state_key(server_label))
        client_id = tool_context.state.get(f"_mcp_oauth_client_id:{server_label}")
        token_endpoint = tool_context.state.get(f"_mcp_oauth_token_endpoint:{server_label}")
        redirect_uri = tool_context.state.get(f"_mcp_oauth_redirect_uri:{server_label}")

        if not verifier:
            return {
                "status": "error",
                "message": (
                    f"No pending OAuth flow found for server '{server_label}'. "
                    "Please call initiate_mcp_oauth() first."
                ),
            }

        # Verify state parameter if provided
        if provided_state and expected_state and not secrets.compare_digest(provided_state, expected_state):
            return {
                "status": "error",
                "message": "OAuth state parameter mismatch. The authorization code may be for a different session.",
            }

        # Exchange authorization code for access token
        try:
            token_data: dict[str, str] = {
                "grant_type": "authorization_code",
                "code": code,
                "client_id": client_id or "",
                "code_verifier": verifier,
                "redirect_uri": redirect_uri or "urn:ietf:wg:oauth:2.0:oob",
            }

            async with httpx.AsyncClient(timeout=30.0, verify=False) as client:
                resp = await client.post(
                    token_endpoint,
                    data=token_data,
                    headers={"Content-Type": "application/x-www-form-urlencoded", "Accept": "application/json"},
                )

            if resp.status_code != 200:
                body = resp.text
                logger.warning("Token exchange failed (%s): %s", resp.status_code, body)
                return {
                    "status": "error",
                    "message": f"Token exchange failed (HTTP {resp.status_code}): {body}",
                }

            token_resp = resp.json()
            access_token = token_resp.get("access_token")
            if not access_token:
                return {
                    "status": "error",
                    "message": f"Token response did not contain access_token: {token_resp}",
                }

            # Store the token using the same key as set_mcp_token — header_provider picks it up automatically
            state_key = mcp_token_state_key(server_label)
            tool_context.state[state_key] = access_token

            # Clean up transient OAuth state
            for key in [
                _pkce_state_key(server_label),
                _oauth_state_key(server_label),
                f"_mcp_oauth_client_id:{server_label}",
                f"_mcp_oauth_token_endpoint:{server_label}",
                f"_mcp_oauth_redirect_uri:{server_label}",
            ]:
                tool_context.state.pop(key, None)

            scope = token_resp.get("scope", "")
            logger.info(
                "complete_mcp_oauth: token stored for server '%s' (scope=%s)",
                server_label,
                scope,
            )
            return {
                "status": "ok",
                "message": (
                    f"Successfully authenticated with '{server_label}'. "
                    f"Scope: {scope or '(default)'}. "
                    "Your subsequent MCP tool calls will use this token automatically."
                ),
            }

        except Exception as e:
            logger.exception("complete_mcp_oauth failed for server '%s'", server_label)
            return {"status": "error", "message": f"Token exchange failed: {e}"}
