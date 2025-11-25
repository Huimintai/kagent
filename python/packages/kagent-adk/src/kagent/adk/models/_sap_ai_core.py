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
            content_items: list[dict] = []
            
            for part in content.parts or []:
                # Handle text parts
                if getattr(part, "text", None):
                    content_items.append({"text": part.text})
                # Handle function calls (tool use)
                elif getattr(part, "function_call", None):
                    func_call = part.function_call
                    content_items.append({
                        "toolUse": {
                            "toolUseId": func_call.id or "",
                            "name": func_call.name,
                            "input": func_call.args
                        }
                    })
                # Handle function responses (tool results)
                elif getattr(part, "function_response", None):
                    func_resp = part.function_response
                    # Extract response content - use model_dump() to properly serialize
                    try:
                        # Try to get the response as a dictionary
                        if hasattr(func_resp.response, 'model_dump'):
                            response_dict = func_resp.response.model_dump(by_alias=True, exclude_none=True)
                            result_text = json.dumps(response_dict)
                        elif isinstance(func_resp.response, dict):
                            result_text = json.dumps(func_resp.response)
                        else:
                            result_text = str(func_resp.response)
                    except Exception as e:
                        logger.warning(f"Failed to serialize function response: {e}")
                        result_text = str(func_resp.response)
                    
                    content_items.append({
                        "toolResult": {
                            "toolUseId": func_resp.id or "",
                            "content": [{"text": result_text}],
                            "status": "success"  # Assume success; could check for error markers
                        }
                    })
            
            if content_items:
                messages.append({
                    "role": role,
                    "content": content_items
                })
        return messages

    def _convert_response_to_llm_response(self, response_data: dict) -> LlmResponse:
        """Convert SAP AI Core response (Converse API or OpenAI-style) to LlmResponse."""
        # Log raw response for debugging
        logger.info(f"Raw API response keys: {list(response_data.keys())}")
        logger.debug(f"Raw API response: {json.dumps(response_data, indent=2)[:2000]}")
        
        # First try Converse style (as per sample response)
        try:
            if "output" in response_data and isinstance(response_data.get("output"), dict):
                output = response_data["output"]
                if "message" in output and isinstance(output["message"], dict):
                    message = output["message"]
                    role = message.get("role", "assistant")
                    content_items = message.get("content", [])
                    
                    # Process content items (text and tool calls)
                    parts: list[types.Part] = []
                    for item in content_items:
                        if isinstance(item, dict):
                            # Handle text content
                            if "text" in item and isinstance(item["text"], str):
                                parts.append(types.Part.from_text(text=item["text"]))
                            # Handle tool use (function calls)
                            elif "toolUse" in item and isinstance(item["toolUse"], dict):
                                tool_use = item["toolUse"]
                                function_call = types.FunctionCall(
                                    name=tool_use.get("name", ""),
                                    args=tool_use.get("input", {}),
                                    id=tool_use.get("toolUseId", "")
                                )
                                parts.append(types.Part(function_call=function_call))
                    
                    # If no parts extracted, add empty text
                    if not parts:
                        parts = [types.Part.from_text(text="")]
                    
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
                    # Note: tool_use is still STOP - the framework detects tool calls via Part.function_call

                    logger.info(f"Successfully parsed Converse response, finish_reason: {finish_reason}, parts: {len(parts)}")
                    return LlmResponse(
                        content=content,
                        usage_metadata=usage_metadata,
                        finish_reason=finish_reason
                    )
        except Exception as e:
            # Fall through to OpenAI-style parsing
            logger.warning(f"Failed to parse Converse API response: {e}", exc_info=True)

        # Fallback: OpenAI-style response
        choices = response_data.get("choices", [])
        if not choices:
            logger.error(
                f"Unable to parse response in either Converse or OpenAI format. "
                f"Response keys: {list(response_data.keys())}, "
                f"Full response: {json.dumps(response_data, indent=2)[:1000]}"
            )
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

    def _normalize_json_schema(self, schema: dict) -> dict:
        """Normalize JSON schema to comply with JSON Schema Draft 2020-12.
        
        Fixes common issues:
        - Converts snake_case keys (any_of, one_of) to camelCase (anyOf, oneOf)
        - Converts uppercase type values (STRING, INTEGER) to lowercase
        - Removes nested type wrappers
        - Converts nullable patterns to proper type arrays
        """
        if not isinstance(schema, dict):
            return schema
        
        normalized = {}
        
        for key, value in schema.items():
            # Normalize key names
            if key == "any_of":
                key = "anyOf"
            elif key == "one_of":
                key = "oneOf"
            elif key == "all_of":
                key = "allOf"
            
            # Handle type field
            if key == "type":
                if isinstance(value, str):
                    # Convert to lowercase
                    value = value.lower()
                    # Map genai types to JSON Schema types
                    type_map = {
                        "string": "string",
                        "integer": "integer",
                        "number": "number",
                        "boolean": "boolean",
                        "object": "object",
                        "array": "array",
                        "null": "null"
                    }
                    value = type_map.get(value, "string")
            
            # Recursively normalize nested structures
            if isinstance(value, dict):
                value = self._normalize_json_schema(value)
            elif isinstance(value, list):
                value = [self._normalize_json_schema(item) if isinstance(item, dict) else item for item in value]
            
            normalized[key] = value
        
        # Handle nullable + anyOf patterns - simplify to type array
        if "anyOf" in normalized:
            any_of = normalized["anyOf"]
            # Check if it's a nullable pattern like [{type: "string"}, {type: "null"}] or [{type: "string"}, {nullable: true}]
            if isinstance(any_of, list):
                types_in_any_of = []
                has_nullable = False
                primary_type = None
                
                for item in any_of:
                    if isinstance(item, dict):
                        if "type" in item:
                            item_type = item["type"]
                            if item_type == "null":
                                has_nullable = True
                            elif item_type in ["string", "integer", "number", "boolean", "array"]:
                                primary_type = item_type
                            # Skip "object" type for null union - it's likely a pydantic artifact
                        elif item.get("nullable"):
                            has_nullable = True
                
                # If we have a primary type + nullable indicator, convert to type array
                if primary_type and has_nullable:
                    del normalized["anyOf"]
                    normalized["type"] = [primary_type, "null"]
                # If all are simple types without extra properties, keep as array
                elif len(any_of) == 2 and all(isinstance(item, dict) and "type" in item and item["type"] in ["string", "integer", "number", "boolean", "null"] and len(item) == 1 for item in any_of):
                    del normalized["anyOf"]
                    normalized["type"] = [item["type"] for item in any_of]
        
        # Remove invalid/redundant fields
        if "nullable" in normalized:
            del normalized["nullable"]
        
        # Remove duplicate type specifications
        if "type" in normalized and isinstance(normalized["type"], str) and "anyOf" in normalized:
            # Keep anyOf, remove simple type
            del normalized["type"]
        
        return normalized
    
    def _convert_tools_to_converse(self, tools: list[types.Tool]) -> list[dict]:
        """Convert genai Tools to AWS Bedrock Converse API format.
        
        Converse API tool format:
        {
            "toolSpec": {
                "name": "tool_name",
                "description": "tool description",
                "inputSchema": {
                    "json": {
                        "type": "object",
                        "properties": {...},
                        "required": [...]
                    }
                }
            }
        }
        """
        converse_tools = []
        
        for tool in tools:
            if tool.function_declarations:
                for func_decl in tool.function_declarations:
                    # Build input schema
                    properties = {}
                    required = []
                    
                    if func_decl.parameters:
                        if func_decl.parameters.properties:
                            for prop_name, prop_schema in func_decl.parameters.properties.items():
                                # Convert schema to dict, handling genai types
                                prop_dict = prop_schema.model_dump(exclude_none=True)
                                # Normalize to JSON Schema Draft 2020-12
                                normalized_prop = self._normalize_json_schema(prop_dict)
                                properties[prop_name] = normalized_prop
                        
                        if func_decl.parameters.required:
                            required = func_decl.parameters.required
                    
                    # Build tool spec
                    tool_spec = {
                        "toolSpec": {
                            "name": func_decl.name or "",
                            "description": func_decl.description or "",
                            "inputSchema": {
                                "json": {
                                    "type": "object",
                                    "properties": properties,
                                    "required": required
                                }
                            }
                        }
                    }
                    converse_tools.append(tool_spec)
        
        return converse_tools

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
        
        # Handle tools - Convert genai tools to Converse API format
        if llm_request.config and llm_request.config.tools:
            genai_tools = []
            for tool in llm_request.config.tools:
                if hasattr(tool, "function_declarations"):
                    genai_tools.append(tool)
            
            if genai_tools:
                converse_tools = self._convert_tools_to_converse(genai_tools)
                if converse_tools:
                    payload["toolConfig"] = {"tools": converse_tools}
                    logger.info(f"Added {len(converse_tools)} tools to request")
                    # Log first tool for debugging (at debug level)
                    if converse_tools:
                        logger.info(f"Sample tool schema: {json.dumps(converse_tools[0], indent=2)}")
        
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
                logger.info(f"Sending POST request to {endpoint}")
                response = await self._client.post(endpoint, json=payload, headers=headers)
                logger.info(f"Received response with status {response.status_code}")
                response.raise_for_status()
                response_data = response.json()
                logger.info(f"Parsed response data, keys: {list(response_data.keys())}")
                llm_response = self._convert_response_to_llm_response(response_data)
                logger.info(f"Converted to LlmResponse, error_code: {llm_response.error_code}, finish_reason: {llm_response.finish_reason}")
                yield llm_response
        
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
            logger.error(f"Unexpected error in generate_content_async: {e}", exc_info=True)
            yield LlmResponse(error_code="API_ERROR", error_message=str(e))

