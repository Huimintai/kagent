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

"""Tests for SAP AI Core model adapter."""

import pytest
from unittest.mock import Mock, patch, AsyncMock
from kagent.adk.models._sapaicore import SAPAICoreModel


@pytest.fixture
def mock_token_fetcher():
    """Create mock token fetcher for tests."""
    with patch("kagent.adk.models._sapaicore.create_token_fetcher") as mock_token_fetcher:
        mock_get_token = Mock(return_value="Bearer test-token")
        mock_token_fetcher.return_value = (mock_get_token, "https://api.example.com", "default")
        yield mock_token_fetcher


@pytest.fixture
def sap_model(mock_token_fetcher):
    """Create test SAP AI Core model instance."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        resource_group="default",
        deployment_id="d123",
    )
    return model


@pytest.mark.asyncio
async def test_deployment_id_resolution(sap_model):
    """Test deployment ID is used when provided."""
    deployment_id = await sap_model._resolve_deployment_id()
    assert deployment_id == "d123"


@pytest.mark.asyncio
async def test_initialization(mock_token_fetcher):
    """Test token fetcher created on init."""
    model = SAPAICoreModel(
        model="gpt-4o",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    # Verify token fetcher was created with correct params
    mock_token_fetcher.assert_called_once()
    call_kwargs = mock_token_fetcher.call_args.kwargs
    assert call_kwargs["base_url"] == "https://api.example.com"
    assert call_kwargs["token_url"] == "https://auth.example.com"
    assert call_kwargs["resource_group"] == "default"


@pytest.mark.asyncio
async def test_initialization_custom_resource_group(mock_token_fetcher):
    """Test token fetcher created with custom resource group."""
    model = SAPAICoreModel(
        model="gpt-4o",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        resource_group="custom-group",
    )

    # Verify token fetcher was created with custom resource group
    call_kwargs = mock_token_fetcher.call_args.kwargs
    assert call_kwargs["resource_group"] == "custom-group"


@pytest.mark.asyncio
async def test_deployment_id_caching(mock_token_fetcher):
    """Test deployment ID is cached after first query."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        deployment_id=None,  # Not provided
    )

    # Mock the query method
    model._query_deployment_id = AsyncMock(return_value="d456")

    # First call should query
    deployment_id_1 = await model._resolve_deployment_id()
    assert deployment_id_1 == "d456"
    assert model._query_deployment_id.call_count == 1

    # Second call should use cache
    deployment_id_2 = await model._resolve_deployment_id()
    assert deployment_id_2 == "d456"
    assert model._query_deployment_id.call_count == 1  # No additional call


@pytest.mark.asyncio
async def test_query_deployment_id_success(sap_model):
    """Test successful deployment ID query from API."""
    # Mock HTTP client
    mock_http_client = Mock()
    mock_response = Mock()
    mock_response.json.return_value = {
        "resources": [
            {
                "id": "d789",
                "targetStatus": "RUNNING",
                "details": {
                    "resources": {
                        "backend_details": {
                            "model": {
                                "name": "gpt-4",
                                "version": "latest",
                            }
                        }
                    }
                },
            }
        ]
    }
    mock_response.raise_for_status = Mock()
    mock_http_client.get.return_value = mock_response

    sap_model._http_client = mock_http_client

    deployment_id = await sap_model._query_deployment_id("Bearer test-token")

    assert deployment_id == "d789"

    # Verify request parameters
    call_args = mock_http_client.get.call_args
    assert "v2/lm/deployments" in call_args[0][0]
    assert call_args[1]["headers"]["Authorization"] == "Bearer test-token"
    assert call_args[1]["headers"]["AI-Resource-Group"] == "default"


@pytest.mark.asyncio
async def test_query_deployment_id_no_running_deployments(sap_model):
    """Test error when no running deployments found."""
    # Mock HTTP client with no running deployments
    mock_http_client = Mock()
    mock_response = Mock()
    mock_response.json.return_value = {
        "resources": [
            {
                "id": "d999",
                "targetStatus": "STOPPED",
                "details": {
                    "resources": {
                        "backend_details": {
                            "model": {
                                "name": "gpt-4",
                                "version": "latest",
                            }
                        }
                    }
                },
            }
        ]
    }
    mock_response.raise_for_status = Mock()
    mock_http_client.get.return_value = mock_response

    sap_model._http_client = mock_http_client

    with pytest.raises(RuntimeError, match="No running deployment found"):
        await sap_model._query_deployment_id("Bearer test-token")


@pytest.mark.asyncio
async def test_query_deployment_id_model_not_found(sap_model):
    """Test error when model not found in deployments."""
    # Mock HTTP client with different model
    mock_http_client = Mock()
    mock_response = Mock()
    mock_response.json.return_value = {
        "resources": [
            {
                "id": "d888",
                "targetStatus": "RUNNING",
                "details": {
                    "resources": {
                        "backend_details": {
                            "model": {
                                "name": "gpt-3.5-turbo",
                                "version": "latest",
                            }
                        }
                    }
                },
            }
        ]
    }
    mock_response.raise_for_status = Mock()
    mock_http_client.get.return_value = mock_response

    sap_model._http_client = mock_http_client

    with pytest.raises(RuntimeError, match="No running deployment found"):
        await sap_model._query_deployment_id("Bearer test-token")


@pytest.mark.asyncio
async def test_query_deployment_id_version_matching(mock_token_fetcher):
    """Test deployment ID query with version matching."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        model_version="0613",
        deployment_id=None,
    )

    # Mock HTTP client with multiple versions
    mock_http_client = Mock()
    mock_response = Mock()
    mock_response.json.return_value = {
        "resources": [
            {
                "id": "d-latest",
                "targetStatus": "RUNNING",
                "details": {
                    "resources": {
                        "backend_details": {
                            "model": {
                                "name": "gpt-4",
                                "version": "latest",
                            }
                        }
                    }
                },
            },
            {
                "id": "d-0613",
                "targetStatus": "RUNNING",
                "details": {
                    "resources": {
                        "backend_details": {
                            "model": {
                                "name": "gpt-4",
                                "version": "0613",
                            }
                        }
                    }
                },
            },
        ]
    }
    mock_response.raise_for_status = Mock()
    mock_http_client.get.return_value = mock_response

    model._http_client = mock_http_client

    deployment_id = await model._query_deployment_id("Bearer test-token")

    # Should match the specific version
    assert deployment_id == "d-0613"


@pytest.mark.asyncio
async def test_query_deployment_id_http_error(sap_model):
    """Test error handling for HTTP errors during deployment query."""
    import httpx

    # Mock HTTP client that raises error
    mock_http_client = Mock()
    mock_response = Mock()
    mock_response.status_code = 401
    mock_response.text = "Unauthorized"
    mock_http_client.get.side_effect = httpx.HTTPStatusError(
        "401 Unauthorized", request=Mock(), response=mock_response
    )

    sap_model._http_client = mock_http_client

    with pytest.raises(RuntimeError, match="Failed to query deployments"):
        await sap_model._query_deployment_id("Bearer test-token")


@pytest.mark.asyncio
async def test_get_token(sap_model):
    """Test token retrieval using token getter."""
    token = await sap_model._get_token()
    assert token == "Bearer test-token"


@pytest.mark.asyncio
async def test_supported_models():
    """Test supported models list."""
    models = SAPAICoreModel.supported_models()
    assert len(models) > 0
    assert r"gpt-.*" in models
    assert r"claude-.*" in models
    assert r"gemini-.*" in models


@pytest.mark.asyncio
async def test_tls_config_defaults(mock_token_fetcher):
    """Test default TLS configuration."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    disable_verify, ca_cert_path, disable_system_cas = model._get_tls_config()

    assert disable_verify is False
    assert ca_cert_path is None
    assert disable_system_cas is False


@pytest.mark.asyncio
async def test_tls_config_with_custom_ca(mock_token_fetcher):
    """Test TLS configuration with custom CA certificate."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        tls_ca_cert_path="/etc/ssl/certs/ca.crt",
    )

    disable_verify, ca_cert_path, disable_system_cas = model._get_tls_config()

    assert disable_verify is False
    assert ca_cert_path == "/etc/ssl/certs/ca.crt"
    assert disable_system_cas is False


@pytest.mark.asyncio
async def test_tls_config_disable_verify(mock_token_fetcher):
    """Test TLS configuration with verification disabled."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
        tls_disable_verify=True,
    )

    disable_verify, ca_cert_path, disable_system_cas = model._get_tls_config()

    assert disable_verify is True
    assert ca_cert_path is None
    assert disable_system_cas is False


@pytest.mark.asyncio
async def test_http_client_creation(mock_token_fetcher):
    """Test HTTP client creation with default settings."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    client = model._create_http_client()

    assert client is not None
    assert client.timeout.read == 30.0


@pytest.mark.asyncio
async def test_http_client_with_custom_tls(mock_token_fetcher):
    """Test HTTP client creation with custom TLS configuration."""
    with patch("kagent.adk.models._sapaicore.create_ssl_context") as mock_ssl:
        import ssl

        mock_ssl_context = Mock(spec=ssl.SSLContext)
        mock_ssl.return_value = mock_ssl_context

        model = SAPAICoreModel(
            model="gpt-4",
            base_url="https://api.example.com",
            token_url="https://auth.example.com",
            tls_ca_cert_path="/etc/ssl/certs/ca.crt",
        )

        client = model._create_http_client()

        # Verify create_ssl_context was called
        mock_ssl.assert_called_once_with(
            disable_verify=False,
            ca_cert_path="/etc/ssl/certs/ca.crt",
            disable_system_cas=False,
        )

        assert client is not None


@pytest.mark.asyncio
async def test_async_http_client_creation_no_tls(mock_token_fetcher):
    """Test async HTTP client creation without TLS configuration."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    client = model._create_async_http_client()

    # Should return None when no TLS config
    assert client is None


@pytest.mark.asyncio
async def test_async_http_client_creation_with_tls(mock_token_fetcher):
    """Test async HTTP client creation with TLS configuration."""
    with patch("kagent.adk.models._sapaicore.create_ssl_context") as mock_ssl:
        import ssl

        mock_ssl_context = Mock(spec=ssl.SSLContext)
        mock_ssl.return_value = mock_ssl_context

        model = SAPAICoreModel(
            model="gpt-4",
            base_url="https://api.example.com",
            token_url="https://auth.example.com",
            tls_disable_verify=True,
        )

        client = model._create_async_http_client()

        # Verify create_ssl_context was called
        mock_ssl.assert_called_once_with(
            disable_verify=True,
            ca_cert_path=None,
            disable_system_cas=False,
        )

        assert client is not None


@pytest.mark.asyncio
async def test_model_version_latest_default(mock_token_fetcher):
    """Test that model version defaults to 'latest'."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    assert model.model_version == "latest"


@pytest.mark.asyncio
async def test_client_identifier_default(mock_token_fetcher):
    """Test that client identifier defaults to 'kagent'."""
    model = SAPAICoreModel(
        model="gpt-4",
        base_url="https://api.example.com",
        token_url="https://auth.example.com",
    )

    assert model.client_identifier == "kagent"
