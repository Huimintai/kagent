import { create } from "zustand";

export interface SkillCliPreset {
  label: string;
  ref: string;
  description?: string;
}

export interface AppConfig {
  disableModelCreation: boolean;
  modelCreationDisabledMessage: string;
  disableMcpServerCreation: boolean;
  disableByoAgentCreation: boolean;
  disableSandboxCreation: boolean;
  disableCliContainers: boolean;
  disablePromptLibrary: boolean;
  disableSchedules: boolean;
  allowedNamespace: string | null;
  protectedAgentNames: string[];
  skillCliPresets: SkillCliPreset[];
  loaded: boolean;
}

const DEFAULTS: Omit<AppConfig, "loaded"> = {
  disableModelCreation: false,
  modelCreationDisabledMessage:
    "Model creation is disabled. Please use the pre-defined models provided by your administrator.",
  disableMcpServerCreation: true,
  disableByoAgentCreation: true,
  disableSandboxCreation: true,
  disableCliContainers: true,
  disablePromptLibrary: true,
  disableSchedules: true,
  allowedNamespace: null,
  protectedAgentNames: [],
  skillCliPresets: [],
};

async function fetchConfig(): Promise<Omit<AppConfig, "loaded">> {
  try {
    const res = await fetch("/api/config", { credentials: "include" });
    if (!res.ok) return DEFAULTS;
    return await res.json();
  } catch {
    return DEFAULTS;
  }
}

export const useAppConfig = create<AppConfig>((set) => {
  if (typeof window !== "undefined") {
    void fetchConfig().then((cfg) => set({ ...cfg, loaded: true }));
  }
  return { ...DEFAULTS, loaded: false };
});

export function isAgentProtectedCheck(protectedNames: string[], agentName: string): boolean {
  if (protectedNames.length === 0) return false;
  return protectedNames.includes(agentName.toLowerCase());
}

export function isEffectivelyProtectedCheck(
  protectedNames: string[],
  agentName: string,
  isOwner: boolean
): boolean {
  if (isAgentProtectedCheck(protectedNames, agentName)) return true;
  if (!isOwner) return true;
  return false;
}
