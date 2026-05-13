package aicoreproxy

import "encoding/json"

// anthropicRequest is a minimal representation of an Anthropic Messages API request
// to extract fields needed for routing decisions.
type anthropicRequest struct {
	Model  string `json:"model,omitempty"`
	Stream bool   `json:"stream,omitempty"`
}

// fixAnthropicRequestForBedrock applies minimal transformations needed for
// SAP AI Core's Bedrock-compatible Anthropic endpoint:
// - Strips cache_control.scope fields (not supported by Bedrock)
// - Removes model field (deployment determines the model)
// - Removes unsupported top-level fields (context_management, etc.)
func fixAnthropicRequestForBedrock(body []byte, req *anthropicRequest) []byte {
	// Parse into generic map for manipulation
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return body // Pass through if we can't parse
	}

	// Remove model field — Bedrock deployments are model-specific
	delete(raw, "model")

	// Remove fields not supported by Bedrock's Anthropic endpoint
	delete(raw, "context_management")
	delete(raw, "stream") // Bedrock uses URL path for streaming, not body field

	// Bedrock requires anthropic_version in the request body
	if _, ok := raw["anthropic_version"]; !ok {
		raw["anthropic_version"] = "bedrock-2023-05-31"
	}

	// Strip cache_control.scope from system and messages
	if system, ok := raw["system"]; ok {
		raw["system"] = stripScopeFromContent(system)
	}
	if messages, ok := raw["messages"].([]any); ok {
		for i, msg := range messages {
			if m, ok := msg.(map[string]any); ok {
				if content, exists := m["content"]; exists {
					m["content"] = stripScopeFromContent(content)
					messages[i] = m
				}
			}
		}
		raw["messages"] = messages
	}

	result, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return result
}

// stripScopeFromContent removes the "scope" field from cache_control objects
// in content blocks. Claude Code adds scope fields that Bedrock rejects.
func stripScopeFromContent(content any) any {
	switch c := content.(type) {
	case []any:
		for i, item := range c {
			if block, ok := item.(map[string]any); ok {
				if cc, exists := block["cache_control"].(map[string]any); exists {
					delete(cc, "scope")
					block["cache_control"] = cc
				}
				c[i] = block
			}
		}
		return c
	default:
		return content
	}
}
