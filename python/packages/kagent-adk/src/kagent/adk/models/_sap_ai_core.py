"""SAP AI Core model implementation for KAgent."""

from __future__ import annotations

import json
import logging
import os
from functools import cached_property
from typing import TYPE_CHECKING, AsyncGenerator, Literal, Optional

import httpx
from google.adk.models import BaseLlm
from google.adk.models.llm_response import LlmResponse
from google.genai import types
from pydantic import Field

if TYPE_CHECKING:
    from google.adk.models.llm_request import LlmRequest

logger = logging.getLogger(__name__)


class SAPAICore(BaseLlm):
    """SAP AI Core model implementation.
    
    This adapter enables KAgent to call SAP AI Core generative AI deployments.
    It supports OAuth2 authentication and various LLM parameters.
    
    Attributes:
        model: The model identifier (e.g., "gpt-4", "claude-3")
        base_url: SAP AI Core inference API base URL
        resource_group: SAP AI Core resource group
        deployment_id: SAP AI Core deployment ID
        api_key: Optional API key (or use SAP_AI_CORE_API_KEY env var)
        auth_url: OAuth token endpoint URL
        client_id: OAuth client ID
        client_secret: OAuth client secret (or use SAP_AI_CORE_CLIENT_SECRET env var)
        default_headers: Additional HTTP headers
        temperature: Sampling temperature
        max_tokens: Maximum tokens to generate
        top_p: Top-p sampling parameter
        top_k: Top-k sampling parameter
        frequency_penalty: Frequency penalty
        presence_penalty: Presence penalty
        timeout: Request timeout in seconds
    """

    type: Literal["sap_ai_core"]
    model: str
    base_url: str
    resource_group: str
    deployment_id: str
    api_key: Optional[str] = Field(default=None, exclude=True)
    auth_url: Optional[str] = None
    client_id: Optional[str] = None
    client_secret: Optional[str] = Field(default=None, exclude=True)
    default_headers: Optional[dict[str, str]] = None
    temperature: Optional[str] = None
    max_tokens: Optional[int] = None
    top_p: Optional[str] = None
    top_k: Optional[int] = None
    frequency_penalty: Optional[str] = None
    presence_penalty: Optional[str] = None
    timeout: Optional[int] = 60

    @classmethod
    def supported_models(cls) -> list[str]:
        """Returns a list of supported models in regex for LlmRegistry."""
        # SAP AI Core supports various models through deployments
        return [r".*"]

    def _get_api_key(self) -> Optional[str]:
        """Get API key from parameter or environment variable."""
        return self.api_key or os.environ.get("SAP_AI_CORE_API_KEY")

    @cached_property
    def _client(self) -> httpx.AsyncClient:
        """Get the HTTP client with authentication."""
        headers = self.default_headers.copy() if self.default_headers else {}
        
        # Get API key from parameter or environment
        api_key = self._get_api_key()
        
        if api_key:
            headers["Authorization"] = f"Bearer {api_key}"
        
        # Add SAP AI Core specific headers
        headers["AI-Resource-Group"] = self.resource_group
        headers["Content-Type"] = "application/json"
        
        return httpx.AsyncClient(
            base_url=self.base_url,
            headers=headers,
            timeout=self.timeout,
        )

    def _normalize_oauth_url(self, url: str) -> str:
        """Normalize OAuth token endpoint URL.
        
        If the URL doesn't end with /oauth/token, append it.
        This handles cases where users provide the base authentication URL.
        """
        url = url.rstrip("/")
        if not url.endswith("/oauth/token"):
            url = f"{url}/oauth/token"
        return url

    async def _get_oauth_token(self) -> Optional[str]:
        """Get OAuth2 access token if OAuth is configured."""
        # Check if OAuth is configured (auth_url and client_id must be non-empty)
        if not self.auth_url or not self.client_id:
            return None
        
        # Get client secret from parameter, config, or environment variable
        client_secret = self.client_secret or os.environ.get("SAP_AI_CORE_CLIENT_SECRET")
        if not client_secret:
            logger.warning("OAuth is configured but client_secret is not available")
            return None
        
        # Normalize OAuth token endpoint URL
        token_url = self._normalize_oauth_url(self.auth_url)
        
        try:
            async with httpx.AsyncClient(follow_redirects=False) as client:
                response = await client.post(
                    token_url,
                    data={
                        "grant_type": "client_credentials",
                        "client_id": self.client_id,
                        "client_secret": client_secret,
                    },
                    headers={"Content-Type": "application/x-www-form-urlencoded"},
                )
                
                # Handle redirects (should not happen for OAuth token endpoint)
                if response.is_redirect:
                    redirect_location = response.headers.get("location", "")
                    error_msg = (
                        f"OAuth token endpoint returned redirect (302). "
                        f"This usually means the authUrl is incorrect. "
                        f"URL: {token_url}, Redirect: {redirect_location}"
                    )
                    logger.error(error_msg)
                    return None
                
                response.raise_for_status()
                token_data = response.json()
                access_token = token_data.get("access_token")
                if not access_token:
                    logger.error(f"No access_token in OAuth response: {token_data}")
                    return None
                return access_token
        except httpx.HTTPStatusError as e:
            error_msg = (
                f"OAuth token request failed with status {e.response.status_code}. "
                f"URL: {token_url}, Response: {e.response.text[:200]}"
            )
            logger.error(error_msg)
            return None
        except Exception as e:
            logger.error(f"Failed to get OAuth token: {e}", exc_info=True)
            return None

    def _convert_content_to_messages(
        self, contents: list[types.Content], system_instruction: Optional[str] = None
    ) -> list[dict]:
        """Convert google.genai Content list to SAP AI Core Converse API messages format.

        Expected format (per working curl example):
        {
          "messages": [
            {"role": "user", "content": [{"text": "..."}]}
          ]
        }
        Each message's content must be a list of objects containing a text field.
        """
        messages: list[dict] = []
        # SAP AI Core Converse currently accepts only roles: user, assistant.
        # We fold any system instruction into a leading synthetic user message
        # so that deployment validation passes.
        if system_instruction:
            messages.append({
                "role": "user",
                "content": [{"text": system_instruction}]
            })

        for content in contents:
            role = "assistant" if content.role == "model" else content.role
            text_parts: list[str] = []
            for part in content.parts or []:
                if getattr(part, "text", None):
                    text_parts.append(part.text)
            if text_parts:
                # Merge parts with newlines into a single text block for now
                messages.append({
                    "role": role,
                    "content": [{"text": "\n".join(text_parts)}]
                })
        return messages

    def _convert_response_to_llm_response(self, response_data: dict) -> LlmResponse:
        """Convert SAP AI Core response (Converse API or OpenAI-style) to LlmResponse."""
        # First try Converse style (as per sample response)
        try:
            if "output" in response_data and isinstance(response_data.get("output"), dict):
                output = response_data["output"]
                if "message" in output and isinstance(output["message"], dict):
                    message = output["message"]
                    role = message.get("role", "assistant")
                    content_items = message.get("content", [])
                    # Concatenate all text fields
                    text_parts: list[str] = []
                    for item in content_items:
                        if isinstance(item, dict) and "text" in item and isinstance(item["text"], str):
                            text_parts.append(item["text"])
                    combined_text = "\n".join(text_parts)
                    parts = [types.Part.from_text(text=combined_text)]
                    content = types.Content(role="model", parts=parts)

                    # Usage mapping (token naming differs)
                    usage_metadata = None
                    if "usage" in response_data and isinstance(response_data["usage"], dict):
                        usage = response_data["usage"]
                        usage_metadata = types.GenerateContentResponseUsageMetadata(
                            prompt_token_count=usage.get("inputTokens", usage.get("prompt_tokens", 0)),
                            candidates_token_count=usage.get("outputTokens", usage.get("completion_tokens", 0)),
                            total_token_count=usage.get("totalTokens", usage.get("total_tokens", 0)),
                        )

                    # Finish reason mapping
                    finish_reason = types.FinishReason.STOP
                    stop_reason = response_data.get("stopReason") or response_data.get("stop_reason")
                    if stop_reason == "max_tokens":
                        finish_reason = types.FinishReason.MAX_TOKENS

                    return LlmResponse(
                        content=content,
                        usage_metadata=usage_metadata,
                        finish_reason=finish_reason
                    )
        except Exception:
            # Fall through to OpenAI-style parsing
            pass

        # Fallback: OpenAI-style response
        choices = response_data.get("choices", [])
        if not choices:
            return LlmResponse(error_code="API_ERROR", error_message="No valid response content")
        choice = choices[0]
        message = choice.get("message", {})
        content_text = message.get("content", "")
        parts = [types.Part.from_text(text=content_text)]
        content = types.Content(role="model", parts=parts)
        usage_metadata = None
        if "usage" in response_data:
            usage = response_data["usage"]
            usage_metadata = types.GenerateContentResponseUsageMetadata(
                prompt_token_count=usage.get("prompt_tokens", 0),
                candidates_token_count=usage.get("completion_tokens", 0),
                total_token_count=usage.get("total_tokens", 0),
            )
        finish_reason = types.FinishReason.STOP
        if choice.get("finish_reason") == "length":
            finish_reason = types.FinishReason.MAX_TOKENS
        return LlmResponse(content=content, usage_metadata=usage_metadata, finish_reason=finish_reason)

    async def generate_content_async(
        self, llm_request: LlmRequest, stream: bool = False
    ) -> AsyncGenerator[LlmResponse, None]:
        """Generate content using SAP AI Core API.
        
        Args:
            llm_request: The LLM request containing messages and configuration
            stream: Whether to stream the response (currently not supported)
            
        Yields:
            LlmResponse objects containing the generated content
        """
        # Get OAuth token if configured
        oauth_token = await self._get_oauth_token()
        
        # Check if we have at least one authentication method
        # We check both at runtime to handle cases where env vars are set after client initialization
        api_key = self._get_api_key()
        if not oauth_token and not api_key:
            error_msg = "Neither OAuth token nor API key is available"
            logger.error(error_msg)
            yield LlmResponse(error_code="AUTH_ERROR", error_message=error_msg)
            return
        
        # Build request headers
        # Priority: OAuth token > API key
        # Note: We need to ensure AI-Resource-Group header is included
        headers = {
            "AI-Resource-Group": self.resource_group,
        }
        if oauth_token:
            headers["Authorization"] = f"Bearer {oauth_token}"
        elif api_key:
            # If OAuth token is not available, use API key
            # We set it here to ensure it's available even if _client was initialized without it
            headers["Authorization"] = f"Bearer {api_key}"
        # If neither is available, we've already returned an error above
        
        # Convert messages
        system_instruction = None
        if llm_request.config and llm_request.config.system_instruction:
            if isinstance(llm_request.config.system_instruction, str):
                system_instruction = llm_request.config.system_instruction
            elif hasattr(llm_request.config.system_instruction, "parts"):
                parts = getattr(llm_request.config.system_instruction, "parts", [])
                text_parts = []
                for part in parts:
                    if hasattr(part, "text") and part.text:
                        text_parts.append(part.text)
                system_instruction = "\n".join(text_parts)
        
        messages = self._convert_content_to_messages(llm_request.contents, system_instruction)
        
        # Build request payload for Converse API
        inference_config: dict = {}
        if self.temperature is not None:
            try:
                temp_value = float(self.temperature)
                if 0.0 <= temp_value <= 2.0:
                    inference_config["temperature"] = temp_value
                else:
                    logger.warning(f"Temperature value {temp_value} out of range [0.0, 2.0], skipping")
            except (ValueError, TypeError):
                logger.warning(f"Invalid temperature value: {self.temperature}")
        if self.max_tokens is not None:
            inference_config["maxTokens"] = self.max_tokens
        if self.top_p is not None:
            try:
                top_p_value = float(self.top_p)
                if 0.0 <= top_p_value <= 1.0:
                    inference_config["topP"] = top_p_value
                else:
                    logger.warning(f"Top-p value {top_p_value} out of range [0.0, 1.0], skipping")
            except (ValueError, TypeError):
                logger.warning(f"Invalid top_p value: {self.top_p}")
        if self.top_k is not None:
            inference_config["topK"] = self.top_k
        if self.frequency_penalty is not None:
            try:
                freq_penalty_value = float(self.frequency_penalty)
                inference_config["frequencyPenalty"] = freq_penalty_value
            except (ValueError, TypeError):
                logger.warning(f"Invalid frequency_penalty value: {self.frequency_penalty}")
        if self.presence_penalty is not None:
            try:
                pres_penalty_value = float(self.presence_penalty)
                inference_config["presencePenalty"] = pres_penalty_value
            except (ValueError, TypeError):
                logger.warning(f"Invalid presence_penalty value: {self.presence_penalty}")

        payload = {"messages": messages}
        # If both temperature and topP present and API forbids them together, prefer temperature.
        if "temperature" in inference_config and "topP" in inference_config:
            logger.warning("Both temperature and topP specified; removing topP to satisfy model constraints.")
            inference_config.pop("topP", None)
        if inference_config:
            payload["inferenceConfig"] = inference_config
        # Do NOT include model field; deployment id already determines model. Including it may trigger 400 errors.
        
        # SAP AI Core inference endpoint
        endpoint = f"/v2/inference/deployments/{self.deployment_id}/converse"
        
        # Log request details for debugging (at debug level)
        logger.debug(f"SAP AI Core API request - URL: {self.base_url}{endpoint}")
        logger.debug(f"SAP AI Core API request - Headers: {headers}")
        logger.debug(f"SAP AI Core API request - Payload: {json.dumps(payload, indent=2)}")
        
        try:
            if stream:
                # Streaming support (if SAP AI Core supports it)
                payload["stream"] = True
                async with self._client.stream("POST", endpoint, json=payload, headers=headers) as response:
                    response.raise_for_status()
                    async for line in response.aiter_lines():
                        if line.startswith("data: "):
                            data = line[6:]
                            if data.strip() == "[DONE]":
                                break
                            try:
                                chunk = json.loads(data)
                                choices = chunk.get("choices", [])
                                if choices and choices[0].get("delta", {}).get("content"):
                                    content_text = choices[0]["delta"]["content"]
                                    content = types.Content(
                                        role="model",
                                        parts=[types.Part.from_text(text=content_text)]
                                    )
                                    yield LlmResponse(
                                        content=content,
                                        partial=True,
                                        turn_complete=choices[0].get("finish_reason") is not None
                                    )
                            except json.JSONDecodeError:
                                continue
            else:
                # Non-streaming request
                response = await self._client.post(endpoint, json=payload, headers=headers)
                response.raise_for_status()
                response_data = response.json()
                yield self._convert_response_to_llm_response(response_data)
        
        except httpx.HTTPStatusError as e:
            # Log detailed error information for debugging
            try:
                error_body = e.response.text
                # Try to parse JSON error response
                try:
                    error_json = e.response.json()
                    error_details = json.dumps(error_json, indent=2)
                except (json.JSONDecodeError, ValueError):
                    error_details = error_body[:1000]
                
                # Log at INFO level so it's visible in production logs
                logger.error(
                    f"SAP AI Core API request failed with {e.response.status_code}:\n"
                    f"URL: {self.base_url}{endpoint}\n"
                    f"Error Response: {error_details}\n"
                    f"Request Payload: {json.dumps(payload, indent=2)}\n"
                    f"Request Headers: {json.dumps({k: v if k != 'Authorization' else 'Bearer ***' for k, v in headers.items()}, indent=2)}"
                )
            except Exception as log_err:
                logger.error(f"Failed to log error details: {log_err}")
            
            # Extract error message from response
            try:
                error_json = e.response.json()
                error_msg = error_json.get("error", {}).get("message", str(error_json))
            except (json.JSONDecodeError, ValueError, AttributeError):
                error_msg = e.response.text[:500] if e.response.text else "Unknown error"
            
            full_error_msg = f"HTTP {e.response.status_code}: {error_msg}"
            yield LlmResponse(error_code="HTTP_ERROR", error_message=full_error_msg)
        except Exception as e:
            yield LlmResponse(error_code="API_ERROR", error_message=str(e))

