// UI Restriction Configuration
//
// Runtime values are served by /api/config and consumed via useAppConfig() hook.
// Server-side code (actions) can read process.env.KAGENT_* directly.
// The NEXT_PUBLIC_* build args are kept as fallbacks for backward compatibility.

export {
  useAppConfig,
  isAgentProtectedCheck,
  isEffectivelyProtectedCheck,
} from "./configStore";
export type { AppConfig } from "./configStore";

// Legacy build-time constants kept for backward compatibility with components
// that haven't migrated to useAppConfig() yet.
export const DISABLE_MODEL_CREATION: boolean =
  process.env.NEXT_PUBLIC_DISABLE_MODEL_CREATION === "true";

export const MODEL_CREATION_DISABLED_MESSAGE: string =
  process.env.NEXT_PUBLIC_MODEL_CREATION_DISABLED_MESSAGE ||
  "Model creation is disabled. Please use the pre-defined models provided by your administrator.";

export const ALLOWED_NAMESPACE: string | null =
  process.env.NEXT_PUBLIC_ALLOWED_NAMESPACE || null;

export const PROTECTED_AGENT_NAMES: string[] = (
  process.env.NEXT_PUBLIC_PROTECTED_AGENTS || ""
)
  .split(",")
  .map((s) => s.trim().toLowerCase())
  .filter(Boolean);

export function isAgentProtected(agentName: string): boolean {
  if (PROTECTED_AGENT_NAMES.length === 0) return false;
  return PROTECTED_AGENT_NAMES.includes(agentName.toLowerCase());
}
