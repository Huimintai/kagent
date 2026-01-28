"""SAP AI Core OAuth2 credential management with token caching."""

from __future__ import annotations

import logging
import os
from datetime import datetime, timedelta, timezone
from threading import Lock
from typing import Callable, Optional

import httpx

logger = logging.getLogger(__name__)


def fetch_credentials(**overrides) -> dict[str, str]:
    """Fetch SAP AI Core credentials from environment variables.

    Resolution order per key:
    1. Explicit kwargs (overrides)
    2. Environment variables (SAP_AI_CORE_*)
    3. Defaults

    Args:
        overrides: Explicit credential overrides (client_id, client_secret, token_url, base_url, resource_group)

    Returns:
        Dictionary with keys: client_id, client_secret, token_url, base_url, resource_group

    Raises:
        ValueError: If required credentials are missing
    """
    creds = {
        "client_id": overrides.get("client_id") or os.getenv("SAP_AI_CORE_CLIENT_ID"),
        "client_secret": overrides.get("client_secret") or os.getenv("SAP_AI_CORE_CLIENT_SECRET"),
        "token_url": overrides.get("token_url") or os.getenv("SAP_AI_CORE_TOKEN_URL"),
        "base_url": overrides.get("base_url") or os.getenv("SAP_AI_CORE_BASE_URL"),
        "resource_group": overrides.get("resource_group") or os.getenv("SAP_AI_CORE_RESOURCE_GROUP", "default"),
    }

    # Validate required fields
    required = ["client_id", "client_secret", "token_url", "base_url"]
    missing = [k for k in required if not creds.get(k)]
    if missing:
        raise ValueError(f"Missing required SAP AI Core credentials: {missing}")

    # Ensure token_url ends with /oauth/token
    token_url = creds["token_url"].rstrip("/")
    if not token_url.endswith("/oauth/token"):
        token_url += "/oauth/token"
    creds["token_url"] = token_url

    return creds


def create_token_fetcher(
    timeout: float = 30.0,
    expiry_buffer_seconds: int = 60,
    **credential_overrides,
) -> tuple[Callable[[httpx.Client], str], str, str]:
    """Create a callable that fetches and caches OAuth2 bearer tokens.

    The returned callable:
      - Automatically refreshes tokens when expired or near expiry
      - Caches tokens thread-safely with configurable refresh buffer
      - Returns tokens with "Bearer " prefix included

    Args:
        timeout: HTTP request timeout in seconds (default: 30.0)
        expiry_buffer_seconds: Refresh token this many seconds before expiry (default: 60)
        credential_overrides: Override credential values (client_id, client_secret, token_url, base_url, resource_group)

    Returns:
        Tuple of (token_getter_func, base_url, resource_group)
        - token_getter_func: Callable[[httpx.Client], str] that returns a valid "Bearer <token>" string
        - base_url: SAP AI Core API base URL
        - resource_group: Resource group identifier

    Raises:
        ValueError: If required credentials are missing
        RuntimeError: If token request fails
    """
    creds = fetch_credentials(**credential_overrides)

    client_id = creds["client_id"]
    client_secret = creds["client_secret"]
    token_url = creds["token_url"]
    base_url = creds["base_url"]
    resource_group = creds["resource_group"]

    # Token cache
    lock = Lock()
    token: Optional[str] = None
    token_expiry: Optional[datetime] = None

    def _request_token(http_client: httpx.Client) -> tuple[str, datetime]:
        """Request new token from OAuth2 endpoint.

        Args:
            http_client: httpx.Client instance for making HTTP requests

        Returns:
            Tuple of (bearer_token, expiry_datetime)

        Raises:
            RuntimeError: If token request fails
        """
        data = {
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
        }

        try:
            resp = http_client.post(
                token_url,
                data=data,
                headers={"Content-Type": "application/x-www-form-urlencoded"},
                timeout=timeout,
            )
            resp.raise_for_status()

            payload = resp.json()
            access_token = payload["access_token"]
            expires_in = int(payload.get("expires_in", 3600))
            expiry_date = datetime.now(timezone.utc) + timedelta(seconds=expires_in)

            logger.debug(f"SAP AI Core token obtained, expires in {expires_in}s")
            return f"Bearer {access_token}", expiry_date

        except Exception as e:
            msg = getattr(resp, "text", str(e)) if "resp" in locals() else str(e)
            raise RuntimeError(f"SAP AI Core token request failed: {msg}") from e

    def get_token(http_client: httpx.Client) -> str:
        """Get valid access token, refreshing if necessary.

        Args:
            http_client: httpx.Client instance for making HTTP requests

        Returns:
            Valid bearer token string with "Bearer " prefix
        """
        nonlocal token, token_expiry

        with lock:
            now = datetime.now(timezone.utc)
            # Refresh if missing, expired, or within buffer window
            if (
                token is None
                or token_expiry is None
                or token_expiry - now < timedelta(seconds=expiry_buffer_seconds)
            ):
                token, token_expiry = _request_token(http_client)
            return token

    return get_token, base_url, resource_group
