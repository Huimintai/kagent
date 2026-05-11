// Model-related constants
export const OLLAMA_DEFAULT_TAG = "latest";
export const OLLAMA_DEFAULT_HOST = "localhost:11434";

// Agent classification labels
export const LABEL_TOOL_TYPE = "kagent.dev/tool-type";
export const LABEL_HAS_MCP = "kagent.dev/has-mcp";
export const LABEL_HAS_SKILL = "kagent.dev/has-skill";
export const LABEL_CATEGORY = "kagent.dev/category";

export type PrivacyFilter = "all" | "my";
