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
