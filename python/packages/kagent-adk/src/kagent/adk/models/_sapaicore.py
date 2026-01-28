"""SAP AI Core model adapter using OpenAI SDK with OAuth2 authentication."""

from __future__ import annotations

import json
import logging
from functools import cached_property
from typing import TYPE_CHECKING, Any, AsyncGenerator, Literal, Optional

import httpx
from google.adk.models import BaseLlm
from google.adk.models.llm_response import LlmResponse
from google.genai import types
from openai import AsyncOpenAI, DefaultAsyncHttpxClient
from pydantic import Field, PrivateAttr

from ._openai import (
    _convert_content_to_openai_messages,
    _convert_openai_response_to_llm_response,
    _convert_tools_to_openai,
)
from ._sap_credentials import create_token_fetcher
from ._ssl import create_ssl_context

if TYPE_CHECKING:
    from google.adk.models.llm_request import LlmRequest

logger = logging.getLogger(__name__)


class SAPAICoreModel(BaseLlm):
    """SAP AI Core model adapter using OpenAI-compatible API with OAuth2 authentication.

    This adapter integrates SAP AI Core's LLM deployments with the ADK framework:
    - Automatic OAuth2 token management (client credentials flow)
    - Dynamic deployment ID resolution if not provided
    - OpenAI SDK compatibility layer
    - Custom TLS/SSL configuration support
    - SAP-specific headers (AI-Resource-Group, AI-Client-Type)

    Required environment variables (if not passed explicitly):
        SAP_AI_CORE_CLIENT_ID: OAuth2 client ID
        SAP_AI_CORE_CLIENT_SECRET: OAuth2 client secret
        SAP_AI_CORE_TOKEN_URL: OAuth2 token endpoint
        SAP_AI_CORE_BASE_URL: SAP AI Core API base URL
        SAP_AI_CORE_RESOURCE_GROUP: Resource group (default: "default")
    """

    model: str
    base_url: str
    token_url: str
    resource_group: str = "default"
    deployment_id: Optional[str] = None
    model_version: str = "latest"
    client_identifier: str = "kagent"
    default_headers: Optional[dict[str, str]] = None

    # TLS/SSL configuration fields
    tls_disable_verify: Optional[bool] = None
    tls_ca_cert_path: Optional[str] = None
    tls_disable_system_cas: Optional[bool] = None

    # Private fields for caching
    _token_getter: Optional[Any] = PrivateAttr(default=None)
    _http_client: Optional[httpx.Client] = PrivateAttr(default=None)
    _resolved_deployment_id: Optional[str] = PrivateAttr(default=None)

    def __init__(self, **data):
        """Initialize SAP AI Core model adapter."""
        super().__init__(**data)

        # Initialize token fetcher with credential overrides
        self._token_getter, _, _ = create_token_fetcher(
            base_url=self.base_url,
            token_url=self.token_url,
            resource_group=self.resource_group,
        )

    @classmethod
    def supported_models(cls) -> list[str]:
        """Returns a list of supported models in regex for LlmRegistry."""
        # SAP AI Core supports various models - match common patterns
        return [
            r"gpt-.*",
            r"claude-.*",
            r"gemini-.*",
            r"anthropic--.*",
            r"meta--.*",
            r"mistral--.*",
            r".*",  # Allow any model name since SAP supports custom deployments
        ]

    def _get_tls_config(self) -> tuple[bool, Optional[str], bool]:
        """Read TLS configuration from instance fields.

        Returns:
            Tuple of (disable_verify, ca_cert_path, disable_system_cas)
        """
        disable_verify = self.tls_disable_verify or False
        ca_cert_path = self.tls_ca_cert_path
        disable_system_cas = self.tls_disable_system_cas or False

        return disable_verify, ca_cert_path, disable_system_cas

    def _create_http_client(self) -> httpx.Client:
        """Create HTTP client with custom SSL context.

        Returns:
            httpx.Client with SSL configuration
        """
        disable_verify, ca_cert_path, disable_system_cas = self._get_tls_config()

        if disable_verify or ca_cert_path or disable_system_cas:
            ssl_context = create_ssl_context(
                disable_verify=disable_verify,
                ca_cert_path=ca_cert_path,
                disable_system_cas=disable_system_cas,
            )
            return httpx.Client(verify=ssl_context, timeout=30.0)

        return httpx.Client(timeout=30.0)

    async def _get_token(self) -> str:
        """Get valid OAuth2 access token.

        Returns:
            Bearer token string (with "Bearer " prefix)
        """
        if self._http_client is None:
            self._http_client = self._create_http_client()

        return self._token_getter(self._http_client)

    async def _query_deployment_id(self, token: str) -> str:
        """Query SAP AI Core API to find deployment ID for the specified model.

        Args:
            token: Valid OAuth2 bearer token (with "Bearer " prefix)

        Returns:
            Deployment ID for the model

        Raises:
            RuntimeError: If no running deployment found for the model
        """
        if self._http_client is None:
            self._http_client = self._create_http_client()

        headers = {
            "Authorization": token,
            "AI-Resource-Group": self.resource_group,
            "Content-Type": "application/json",
            "AI-Client-Type": self.client_identifier,
        }

        url = f"{self.base_url.rstrip('/')}/v2/lm/deployments?$top=10000&$skip=0"

        try:
            response = self._http_client.get(url, headers=headers)
            response.raise_for_status()

            deployments = response.json().get("resources", [])

            # Filter for running deployments with matching model name
            for deployment in deployments:
                if deployment.get("targetStatus") != "RUNNING":
                    continue

                # Extract model name from deployment details
                details = deployment.get("details", {})
                backend_details = details.get("resources", {}).get("backend_details", {})
                model_info = backend_details.get("model", {})

                model_name = model_info.get("name", "")
                model_ver = model_info.get("version", "")

                if not model_name:
                    continue

                # Match model name (case-insensitive, base name without version)
                deployment_base = model_name.split(":")[0].lower()
                requested_base = self.model.split(":")[0].lower()

                if deployment_base == requested_base:
                    # Check version if specified
                    if self.model_version != "latest":
                        if model_ver != self.model_version:
                            continue

                    deployment_id = deployment.get("id")
                    logger.info(
                        f"Found deployment {deployment_id} for model {self.model} (version: {model_ver})"
                    )
                    return deployment_id

            raise RuntimeError(
                f"No running deployment found for model {self.model} "
                f"(version: {self.model_version}) in resource group {self.resource_group}"
            )

        except httpx.HTTPStatusError as e:
            raise RuntimeError(f"Failed to query deployments: {e.response.status_code} {e.response.text}") from e
        except Exception as e:
            raise RuntimeError(f"Failed to query deployments: {e}") from e

    async def _resolve_deployment_id(self) -> str:
        """Resolve deployment ID, using cache or querying API if needed.

        Returns:
            Deployment ID
        """
        # Use explicit deployment_id if provided
        if self.deployment_id:
            return self.deployment_id

        # Use cached resolved deployment_id
        if self._resolved_deployment_id:
            return self._resolved_deployment_id

        # Query API to find deployment
        logger.info(f"Querying deployment ID for model {self.model}")
        token = await self._get_token()
        self._resolved_deployment_id = await self._query_deployment_id(token)
        return self._resolved_deployment_id

    def _create_async_http_client(self) -> Optional[DefaultAsyncHttpxClient]:
        """Create async HTTP client with custom SSL context for OpenAI SDK.

        Returns:
            DefaultAsyncHttpxClient with SSL configuration, or None if no TLS config
        """
        disable_verify, ca_cert_path, disable_system_cas = self._get_tls_config()

        if disable_verify or ca_cert_path or disable_system_cas:
            ssl_context = create_ssl_context(
                disable_verify=disable_verify,
                ca_cert_path=ca_cert_path,
                disable_system_cas=disable_system_cas,
            )
            return DefaultAsyncHttpxClient(verify=ssl_context)

        return None

    async def generate_content_async(
        self, llm_request: LlmRequest, stream: bool = False
    ) -> AsyncGenerator[LlmResponse, None]:
        """Generate content using SAP AI Core LLM deployment.

        Args:
            llm_request: Request containing messages, tools, and configuration
            stream: Whether to stream the response

        Yields:
            LlmResponse objects with generated content
        """
        try:
            # Get fresh token and deployment ID
            token = await self._get_token()
            deployment_id = await self._resolve_deployment_id()

            # Strip "Bearer " prefix - OpenAI SDK adds it automatically
            token_without_prefix = token.replace("Bearer ", "", 1)

            # Construct deployment-specific endpoint
            endpoint = f"{self.base_url.rstrip('/')}/v2/inference/deployments/{deployment_id}"

            # Prepare headers
            headers = {
                "AI-Resource-Group": self.resource_group,
                "AI-Client-Type": self.client_identifier,
            }

            # Add custom headers from config
            if self.default_headers:
                headers.update(self.default_headers)

            # Convert messages
            system_instruction = None
            if llm_request.config and llm_request.config.system_instruction:
                if isinstance(llm_request.config.system_instruction, str):
                    system_instruction = llm_request.config.system_instruction
                elif hasattr(llm_request.config.system_instruction, "parts"):
                    text_parts = []
                    parts = getattr(llm_request.config.system_instruction, "parts", [])
                    if parts:
                        for part in parts:
                            if hasattr(part, "text") and part.text:
                                text_parts.append(part.text)
                        system_instruction = "\n".join(text_parts)

            messages = _convert_content_to_openai_messages(llm_request.contents, system_instruction)

            # Prepare request parameters
            kwargs = {
                "model": llm_request.model or self.model,
                "messages": messages,
            }

            # Handle tools
            if llm_request.config and llm_request.config.tools:
                genai_tools = []
                for tool in llm_request.config.tools:
                    if hasattr(tool, "function_declarations"):
                        genai_tools.append(tool)

                if genai_tools:
                    openai_tools = _convert_tools_to_openai(genai_tools)
                    if openai_tools:
                        kwargs["tools"] = openai_tools
                        kwargs["tool_choice"] = "auto"

            # Create OpenAI client with deployment-specific endpoint
            http_client = self._create_async_http_client()

            client = AsyncOpenAI(
                api_key=token_without_prefix,
                base_url=endpoint,
                default_headers=headers,
                http_client=http_client,
            )

            if stream:
                # Handle streaming
                aggregated_text = ""
                finish_reason = None
                usage_metadata = None
                tool_calls_acc: dict[int, dict[str, Any]] = {}

                async for chunk in await client.chat.completions.create(stream=True, **kwargs):
                    if chunk.choices and chunk.choices[0].delta:
                        delta = chunk.choices[0].delta

                        # Handle text content streaming
                        if delta.content:
                            aggregated_text += delta.content
                            content = types.Content(role="model", parts=[types.Part.from_text(text=delta.content)])
                            yield LlmResponse(
                                content=content, partial=True, turn_complete=chunk.choices[0].finish_reason is not None
                            )

                        # Handle tool call chunks
                        if hasattr(delta, "tool_calls") and delta.tool_calls:
                            for tool_call_chunk in delta.tool_calls:
                                idx = tool_call_chunk.index
                                if idx not in tool_calls_acc:
                                    tool_calls_acc[idx] = {
                                        "id": "",
                                        "name": "",
                                        "arguments": "",
                                    }
                                if tool_call_chunk.id:
                                    tool_calls_acc[idx]["id"] = tool_call_chunk.id
                                if tool_call_chunk.function:
                                    if tool_call_chunk.function.name:
                                        tool_calls_acc[idx]["name"] = tool_call_chunk.function.name
                                    if tool_call_chunk.function.arguments:
                                        tool_calls_acc[idx]["arguments"] += tool_call_chunk.function.arguments

                        if chunk.choices[0].finish_reason:
                            finish_reason = chunk.choices[0].finish_reason

                    if hasattr(chunk, "usage") and chunk.usage:
                        usage_metadata = types.GenerateContentResponseUsageMetadata(
                            prompt_token_count=chunk.usage.prompt_tokens,
                            candidates_token_count=chunk.usage.completion_tokens,
                            total_token_count=chunk.usage.total_tokens,
                        )

                # Yield final aggregated response
                final_parts = []

                if aggregated_text:
                    final_parts.append(types.Part.from_text(text=aggregated_text))

                for idx in sorted(tool_calls_acc.keys()):
                    tc = tool_calls_acc[idx]
                    try:
                        args = json.loads(tc["arguments"]) if tc["arguments"] else {}
                    except json.JSONDecodeError:
                        args = {}

                    part = types.Part.from_function_call(name=tc["name"], args=args)
                    if part.function_call:
                        part.function_call.id = tc["id"]
                    final_parts.append(part)

                # Map finish reason
                final_reason = types.FinishReason.STOP
                if finish_reason == "length":
                    final_reason = types.FinishReason.MAX_TOKENS
                elif finish_reason == "content_filter":
                    final_reason = types.FinishReason.SAFETY
                elif finish_reason == "tool_calls":
                    final_reason = types.FinishReason.STOP

                final_content = types.Content(role="model", parts=final_parts)
                yield LlmResponse(
                    content=final_content,
                    partial=False,
                    finish_reason=final_reason,
                    usage_metadata=usage_metadata,
                    turn_complete=True,
                )
            else:
                # Handle non-streaming
                response = await client.chat.completions.create(stream=False, **kwargs)
                yield _convert_openai_response_to_llm_response(response)

        except Exception as e:
            logger.error(f"SAP AI Core API error: {e}", exc_info=True)
            yield LlmResponse(error_code="API_ERROR", error_message=str(e))
