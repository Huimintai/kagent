// UI Restriction Configuration
// All values are read from NEXT_PUBLIC_* environment variables at build time.
// Set them in .env.local, CI pipeline, or as Docker build-args before building.

/**
 * When true, the "New Model" button is disabled and an info banner is shown.
 */
export const DISABLE_MODEL_CREATION: boolean =
  process.env.NEXT_PUBLIC_DISABLE_MODEL_CREATION === "true";

/**
 * Message displayed when model creation is disabled.
 */
export const MODEL_CREATION_DISABLED_MESSAGE: string =
  process.env.NEXT_PUBLIC_MODEL_CREATION_DISABLED_MESSAGE ||
  "Model creation is disabled. Please use the pre-defined models provided by your administrator.";

/**
 * When true, the "New MCP Server" button is disabled.
 * Defaults to disabled when not set or empty. Set to "false" to enable.
 */
export const DISABLE_MCP_SERVER_CREATION: boolean =
  process.env.NEXT_PUBLIC_DISABLE_MCP_SERVER_CREATION !== "false";

/**
 * When true, the BYO agent type option is disabled in the agent creation form.
 * Defaults to disabled when not set or empty. Set to "false" to enable.
 */
export const DISABLE_BYO_AGENT_CREATION: boolean =
  process.env.NEXT_PUBLIC_DISABLE_BYO_AGENT_CREATION !== "false";

/**
 * When set, all namespace selectors are locked to this single value,
 * and only agents/resources in this namespace are visible.
 */
export const ALLOWED_NAMESPACE: string | null =
  process.env.NEXT_PUBLIC_ALLOWED_NAMESPACE || null;

/**
 * Comma-separated list of agent names that cannot be edited or deleted.
 * Matched case-insensitively against agent.metadata.name.
 */
export const PROTECTED_AGENT_NAMES: string[] = (
  process.env.NEXT_PUBLIC_PROTECTED_AGENTS || ""
)
  .split(",")
  .map((s) => s.trim().toLowerCase())
  .filter(Boolean);

/**
 * Check if a given agent name is protected.
 */
export function isAgentProtected(agentName: string): boolean {
  if (PROTECTED_AGENT_NAMES.length === 0) return false;
  return PROTECTED_AGENT_NAMES.includes(agentName.toLowerCase());
}

/**
 * Check if an agent is effectively protected (immutable) for the current user.
 * Protected if: statically protected by env var, OR the current user is not the owner.
 */
export function isEffectivelyProtected(agentName: string, isOwner: boolean): boolean {
  if (isAgentProtected(agentName)) return true;
  if (!isOwner) return true;
  return false;
}
