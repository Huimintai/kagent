# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Tests for SAP AI Core credential management."""

import pytest
from unittest.mock import Mock, patch
from datetime import datetime, timedelta, timezone
from kagent.adk.models._sap_credentials import fetch_credentials, create_token_fetcher


def test_fetch_credentials_from_env(monkeypatch):
    """Test credential resolution from environment variables."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-client-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    creds = fetch_credentials()

    assert creds["client_id"] == "test-client-id"
    assert creds["client_secret"] == "test-secret"
    assert creds["token_url"] == "https://auth.example.com/oauth/token"
    assert creds["base_url"] == "https://api.example.com"
    assert creds["resource_group"] == "default"


def test_fetch_credentials_missing_required(monkeypatch):
    """Test error when required credentials are missing."""
    monkeypatch.delenv("SAP_AI_CORE_CLIENT_ID", raising=False)
    monkeypatch.delenv("SAP_AI_CORE_CLIENT_SECRET", raising=False)
    monkeypatch.delenv("SAP_AI_CORE_TOKEN_URL", raising=False)
    monkeypatch.delenv("SAP_AI_CORE_BASE_URL", raising=False)

    with pytest.raises(ValueError, match="Missing required SAP AI Core credentials"):
        fetch_credentials()


def test_fetch_credentials_missing_one_required(monkeypatch):
    """Test error when one required credential is missing."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.delenv("SAP_AI_CORE_BASE_URL", raising=False)

    with pytest.raises(ValueError, match="Missing required SAP AI Core credentials"):
        fetch_credentials()


def test_token_url_auto_suffix(monkeypatch):
    """Test that /oauth/token is automatically added to token URL."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com/")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    creds = fetch_credentials()

    assert creds["token_url"] == "https://auth.example.com/oauth/token"


def test_token_url_no_duplicate_suffix(monkeypatch):
    """Test that /oauth/token is not duplicated if already present."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com/oauth/token")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    creds = fetch_credentials()

    assert creds["token_url"] == "https://auth.example.com/oauth/token"


def test_fetch_credentials_with_overrides(monkeypatch):
    """Test credential resolution with explicit overrides."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "env-client-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "env-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://env-auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://env-api.example.com")

    creds = fetch_credentials(
        client_id="override-id",
        base_url="https://override-api.example.com",
    )

    assert creds["client_id"] == "override-id"
    assert creds["client_secret"] == "env-secret"  # Not overridden
    assert creds["base_url"] == "https://override-api.example.com"


def test_fetch_credentials_custom_resource_group(monkeypatch):
    """Test custom resource group configuration."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")
    monkeypatch.setenv("SAP_AI_CORE_RESOURCE_GROUP", "custom-group")

    creds = fetch_credentials()

    assert creds["resource_group"] == "custom-group"


@patch("httpx.Client.post")
def test_token_refresh(mock_post, monkeypatch):
    """Test OAuth2 token refresh logic."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    # Mock successful token response
    mock_response = Mock()
    mock_response.json.return_value = {
        "access_token": "test-token-123",
        "expires_in": 3600,
    }
    mock_response.raise_for_status = Mock()
    mock_post.return_value = mock_response

    get_token, base_url, resource_group = create_token_fetcher()

    import httpx

    http_client = httpx.Client()
    token1 = get_token(http_client)

    assert token1 == "Bearer test-token-123"
    assert base_url == "https://api.example.com"
    assert resource_group == "default"
    assert mock_post.call_count == 1

    # Verify request parameters
    call_args = mock_post.call_args
    assert call_args[0][0] == "https://auth.example.com/oauth/token"
    assert call_args[1]["data"]["grant_type"] == "client_credentials"
    assert call_args[1]["data"]["client_id"] == "test-id"
    assert call_args[1]["data"]["client_secret"] == "test-secret"

    # Second call within expiry should use cached token
    token2 = get_token(http_client)
    assert token2 == "Bearer test-token-123"
    assert mock_post.call_count == 1  # No additional call


@patch("httpx.Client.post")
def test_token_caching(mock_post, monkeypatch):
    """Verify token reuse within expiry window."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    mock_response = Mock()
    mock_response.json.return_value = {
        "access_token": "cached-token",
        "expires_in": 3600,
    }
    mock_response.raise_for_status = Mock()
    mock_post.return_value = mock_response

    get_token, _, _ = create_token_fetcher(expiry_buffer_seconds=60)

    import httpx

    http_client = httpx.Client()

    # First call - should fetch
    token1 = get_token(http_client)
    assert mock_post.call_count == 1

    # Multiple subsequent calls - should use cache
    for _ in range(5):
        token = get_token(http_client)
        assert token == "Bearer cached-token"

    assert mock_post.call_count == 1  # Still only 1 request


@patch("httpx.Client.post")
def test_token_refresh_on_expiry(mock_post, monkeypatch):
    """Test token refresh when near expiry."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    # Mock response with short expiry (less than buffer)
    mock_response = Mock()
    mock_response.json.return_value = {
        "access_token": "short-lived-token",
        "expires_in": 30,  # Less than default 60s buffer
    }
    mock_response.raise_for_status = Mock()
    mock_post.return_value = mock_response

    get_token, _, _ = create_token_fetcher(expiry_buffer_seconds=60)

    import httpx

    http_client = httpx.Client()

    # First call - should fetch
    token1 = get_token(http_client)
    assert token1 == "Bearer short-lived-token"
    assert mock_post.call_count == 1

    # Second call - token is within buffer window, should refresh
    mock_response.json.return_value = {
        "access_token": "refreshed-token",
        "expires_in": 3600,
    }

    token2 = get_token(http_client)
    assert token2 == "Bearer refreshed-token"
    assert mock_post.call_count == 2  # Token was refreshed


@patch("httpx.Client.post")
def test_token_request_failure(mock_post, monkeypatch):
    """Test error handling when token request fails."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    # Mock failed token response
    mock_response = Mock()
    mock_response.raise_for_status.side_effect = Exception("401 Unauthorized")
    mock_response.text = "Invalid credentials"
    mock_post.return_value = mock_response

    get_token, _, _ = create_token_fetcher()

    import httpx

    http_client = httpx.Client()

    with pytest.raises(RuntimeError, match="SAP AI Core token request failed"):
        get_token(http_client)


@patch("httpx.Client.post")
def test_token_missing_access_token(mock_post, monkeypatch):
    """Test error when token response is missing access_token field."""
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "test-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "test-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://api.example.com")

    # Mock response missing access_token
    mock_response = Mock()
    mock_response.json.return_value = {"expires_in": 3600}
    mock_response.raise_for_status = Mock()
    mock_post.return_value = mock_response

    get_token, _, _ = create_token_fetcher()

    import httpx

    http_client = httpx.Client()

    with pytest.raises(RuntimeError):
        get_token(http_client)


def test_create_token_fetcher_with_overrides(monkeypatch):
    """Test token fetcher creation with credential overrides."""
    # Set env vars that should be overridden
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_ID", "env-id")
    monkeypatch.setenv("SAP_AI_CORE_CLIENT_SECRET", "env-secret")
    monkeypatch.setenv("SAP_AI_CORE_TOKEN_URL", "https://env-auth.example.com")
    monkeypatch.setenv("SAP_AI_CORE_BASE_URL", "https://env-api.example.com")

    get_token, base_url, resource_group = create_token_fetcher(
        base_url="https://override-api.example.com",
        resource_group="custom-group",
    )

    # Verify overrides were applied
    assert base_url == "https://override-api.example.com"
    assert resource_group == "custom-group"
