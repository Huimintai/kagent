import { NextResponse } from "next/server";

function envBool(primary: string, fallback: string, defaultValue: boolean): boolean {
  const val = process.env[primary] ?? process.env[fallback];
  if (val === undefined || val === "") return defaultValue;
  // For "opt-in" flags (default false): "true" enables
  // For "opt-out" flags (default true): "false" disables
  if (defaultValue) return val !== "false";
  return val === "true";
}

function envString(primary: string, fallback: string, defaultValue: string): string {
  return process.env[primary] ?? process.env[fallback] ?? defaultValue;
}

export async function GET() {
  const allowedNs = envString("KAGENT_ALLOWED_NAMESPACE", "NEXT_PUBLIC_ALLOWED_NAMESPACE", "");
  const protectedRaw = envString("KAGENT_PROTECTED_AGENTS", "NEXT_PUBLIC_PROTECTED_AGENTS", "");

  return NextResponse.json({
    disableModelCreation: envBool("KAGENT_DISABLE_MODEL_CREATION", "NEXT_PUBLIC_DISABLE_MODEL_CREATION", false),
    modelCreationDisabledMessage: envString(
      "KAGENT_MODEL_CREATION_DISABLED_MESSAGE",
      "NEXT_PUBLIC_MODEL_CREATION_DISABLED_MESSAGE",
      "Model creation is disabled. Please use the pre-defined models provided by your administrator."
    ),
    disableMcpServerCreation: envBool("KAGENT_DISABLE_MCP_SERVER_CREATION", "NEXT_PUBLIC_DISABLE_MCP_SERVER_CREATION", true),
    disableByoAgentCreation: envBool("KAGENT_DISABLE_BYO_AGENT_CREATION", "NEXT_PUBLIC_DISABLE_BYO_AGENT_CREATION", true),
    disablePromptLibrary: envBool("KAGENT_DISABLE_PROMPT_LIBRARY", "NEXT_PUBLIC_DISABLE_PROMPT_LIBRARY", false),
    disableScheduledRuns: envBool("KAGENT_DISABLE_SCHEDULED_RUNS", "NEXT_PUBLIC_DISABLE_SCHEDULED_RUNS", false),
    allowedNamespace: allowedNs || null,
    protectedAgentNames: protectedRaw
      .split(",")
      .map((s) => s.trim().toLowerCase())
      .filter(Boolean),
    skillCliPresets: (() => {
      try {
        return JSON.parse(process.env.KAGENT_SKILL_CLI_PRESETS || "[]");
      } catch {
        return [];
      }
    })(),
  });
}
