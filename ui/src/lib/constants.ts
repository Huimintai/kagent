// Model-related constants
export const OLLAMA_DEFAULT_TAG = "latest";
export const OLLAMA_DEFAULT_HOST = "localhost:11434";

// Agent classification labels
export const LABEL_TOOL_TYPE = "kagent.dev/tool-type";
export const LABEL_ROLE = "kagent.dev/role";
export const LABEL_CATEGORY = "kagent.dev/category";

export const ROLE_OPTIONS = [
  { value: "orchestration", label: "Orchestration Agent" },
  { value: "sub-agent", label: "Sub Agent" },
] as const;

export type PrivacyFilter = "all" | "public" | "my";
