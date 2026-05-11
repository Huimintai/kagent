"use client";
import React, { useState, useEffect, Suspense, useCallback, useMemo } from "react";
import { Brain, Loader2, Settings2, PlusCircle, Trash2, Layers, Package } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formAgentTypeFromApi, formUsesByoSections, formUsesDeclarativeSections } from "@/lib/agentFormLayout";
import { ModelConfig, AgentType, ContextConfig, InlineSkill, DeclarativeRuntime } from "@/types";
import { SystemPromptSection } from "@/components/create/SystemPromptSection";
import { newPromptSourceRow, type PromptSourceRow } from "@/lib/promptSourceRow";
import { ModelSelectionSection } from "@/components/create/ModelSelectionSection";
import { ToolsSection } from "@/components/create/ToolsSection";
import { MemorySection } from "@/components/create/MemorySection";
import { ContextSection } from "@/components/create/ContextSection";
import { useRouter, useSearchParams } from "next/navigation";
import { useAgents } from "@/components/AgentsProvider";
import { LoadingState } from "@/components/LoadingState";
import { ErrorState } from "@/components/ErrorState";
import { AgentFormData } from "@/components/AgentsProvider";
import { Tool, EnvVar } from "@/types";
import { toast } from "sonner";
import { NamespaceCombobox } from "@/components/NamespaceCombobox";
import { CategoryCombobox } from "@/components/CategoryCombobox";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog";
import { useAppConfig, isAgentProtectedCheck } from "@/lib/configStore";
import { LABEL_CATEGORY } from "@/lib/constants";
import type { AgentFormValidationErrors } from "@/components/agent-form/agent-form-types";
import { formRowsToGitRepos, isValidSkillContainerImage as isValidContainerImage, type GitSkillFormRow } from "@/lib/agentSkillsForm";
import { focusFirstFormError } from "@/components/agent-form/focusFirstFormError";
import KagentLogo from "@/components/kagent-logo";
import { FieldRoot, FieldLabel, FieldHint, FieldError, FormSection } from "@/components/agent-form/form-primitives";

const PRIVATE_MODE_ANNOTATION = "kagent.dev/private-mode";

interface ValidationErrors {
  name?: string;
  namespace?: string;
  description?: string;
  type?: string;
  systemPrompt?: string;
  model?: string;
  knowledgeSources?: string;
  tools?: string;
  skills?: string;
  memoryModel?: string;
  memoryTtl?: string;
  serviceAccountName?: string;
  promptSources?: string;
}

interface AgentPageContentProps {
  isEditMode: boolean;
  isViewMode: boolean;
  agentName: string | null;
  agentNamespace: string | null;
}

const DEFAULT_SYSTEM_PROMPT = `You're a helpful agent, made by the kagent team.

# Instructions
    - If user question is unclear, ask for clarification before running any tools
    - Always be helpful and friendly
    - If you don't know how to answer the question DO NOT make things up, tell the user "Sorry, I don't know how to answer that" and ask them to clarify the question further
    - If you are unable to help, or something goes wrong, refer the user to https://kagent.dev for more information or support.

# Response format:
    - ALWAYS format your response as Markdown
    - Your response will include a summary of actions you took and an explanation of the result
    - If you created any artifacts such as files or resources, you will include those in your response as well`

// Inner component that uses useSearchParams, wrapped in Suspense
function AgentPageContent({ isEditMode, isViewMode, agentName, agentNamespace }: AgentPageContentProps) {
  const router = useRouter();
  const { models, loading, error, createNewAgent, updateAgent, getAgent, validateAgentData } = useAgents();
  const { allowedNamespace, disableByoAgentCreation, disableSandboxCreation, disableCliContainers, protectedAgentNames, skillCliPresets } = useAppConfig();

  type SelectedModelType = ModelConfig;

  interface FormState {
    name: string;
    namespace: string;
    description: string;
    privateMode: boolean;
    category: string;
    toolType: string;
    agentType: AgentType;
    systemPrompt: string;
    selectedModel: SelectedModelType | null;
    selectedMemoryModel: SelectedModelType | null;
    memoryTtlDays: string;
    selectedTools: Tool[];
    skillRefs: string[];
    skillGitRepos: GitSkillFormRow[];
    inlineSkills: InlineSkill[];
    byoImage: string;
    byoCmd: string;
    byoArgs: string;
    replicas: string;
    imagePullPolicy: string;
    imagePullSecrets: string[];
    envPairs: { name: string; value?: string; isSecret?: boolean; secretName?: string; secretKey?: string; optional?: boolean }[];
    stream: boolean;
    /** Python vs Go ADK (`spec.declarative.runtime`). */
    declarativeRuntime: DeclarativeRuntime;
    contextConfig: ContextConfig | undefined;
    serviceAccountName: string;
    promptSourceRows: PromptSourceRow[];
    isSubmitting: boolean;
    isLoading: boolean;
    errors: AgentFormValidationErrors;
  }

  const [formDirty, setFormDirty] = useState(false);

  const [state, setState] = useState<FormState>({
    name: "",
    namespace: allowedNamespace || "default",
    description: "",
    privateMode: true,
    category: "",
    toolType: "",
    agentType: "Declarative",
    systemPrompt: isEditMode ? "" : DEFAULT_SYSTEM_PROMPT,
    selectedModel: null,
    selectedMemoryModel: null,
    memoryTtlDays: "",
    selectedTools: [],
    skillRefs: [],
    skillGitRepos: [],
    inlineSkills: [],
    byoImage: "",
    byoCmd: "",
    byoArgs: "",
    replicas: "",
    imagePullPolicy: "",
    imagePullSecrets: [""],
    envPairs: [{ name: "", value: "", isSecret: false }],
    stream: false,
    declarativeRuntime: "python",
    contextConfig: undefined,
    serviceAccountName: "",
    promptSourceRows: [newPromptSourceRow()],
    isSubmitting: false,
    isLoading: isEditMode,
    errors: {},
  });

  const [showPublicConfirm, setShowPublicConfirm] = useState(false);

  // Sync namespace when allowedNamespace loads asynchronously
  useEffect(() => {
    if (allowedNamespace && !isEditMode) {
      setState(prev => prev.namespace === allowedNamespace ? prev : { ...prev, namespace: allowedNamespace });
    }
  }, [allowedNamespace, isEditMode]);

  const useDeclarativeAgentFields = formUsesDeclarativeSections(state.agentType, state.byoImage);
  const showByoFields = formUsesByoSections(state.agentType, state.byoImage);
  const disabled = state.isSubmitting || state.isLoading;

  useEffect(() => {
    if (!formDirty) {
      return;
    }
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => window.removeEventListener("beforeunload", onBeforeUnload);
  }, [formDirty]);

  const resolvedGitSkillRepos = useMemo(
    () => formRowsToGitRepos(state.skillGitRepos || []),
    [state.skillGitRepos],
  );

  const ensureConfigMapSource = useCallback((cmName: string) => {
    const t = cmName.trim();
    if (!t) {
      return;
    }
    setState((prev) => {
      if (prev.promptSourceRows.some((r) => r.name.trim() === t)) {
        return { ...prev, errors: { ...prev.errors, promptSources: undefined } };
      }
      const nonEmpty = prev.promptSourceRows.filter((r) => r.name.trim() !== "");
      return {
        ...prev,
        errors: { ...prev.errors, promptSources: undefined },
        promptSourceRows: [...nonEmpty, { id: crypto.randomUUID(), name: t, alias: "" }],
      };
    });
  }, []);

  const includeSourceIdForConfigMap = useCallback(
    (cmName: string) => {
      const row = state.promptSourceRows.find((r) => r.name.trim() === cmName);
      const a = row?.alias?.trim();
      return a || cmName;
    },
    [state.promptSourceRows],
  );

  const isFormDisabled = state.isSubmitting || state.isLoading || isViewMode;


  // Fetch existing agent data if in edit mode
  useEffect(() => {
    const fetchAgentData = async () => {
      if (isEditMode && agentName && agentNamespace) {
        try {
          setState((prev) => ({ ...prev, isLoading: true }));
          const agentResponse = await getAgent(agentName, agentNamespace);

          if (!agentResponse) {
            toast.error("Agent not found");
            setState((prev) => ({ ...prev, isLoading: false }));
            return;
          }

          if (isAgentProtectedCheck(protectedAgentNames, agentResponse.agent.metadata.name || "")) {
            toast.error("This agent is protected and cannot be edited.");
            router.push("/agents");
            return;
          }

          const agent = agentResponse.agent;
          if (agent) {
            try {
              const baseUpdates: Partial<FormState> = {
                name: agent.metadata.name || "",
                namespace: agent.metadata.namespace || "",
                description: agent.spec?.description || "",
                agentType: formAgentTypeFromApi(agent.spec.type, agentResponse.workloadMode),
                privateMode: agentResponse.private_mode ?? (agent.metadata.annotations?.[PRIVATE_MODE_ANNOTATION] === "true"),
                category: agent.metadata.labels?.[LABEL_CATEGORY] || "",
                toolType: "",
              };
              const useDeclarativeForm = agent.spec.type === "Declarative";
              if (useDeclarativeForm) {
                const decl = agent.spec?.declarative;
                const memorySpec = decl?.memory ?? (agent.spec as { memory?: { modelConfig: string; ttlDays?: number } })?.memory;
                const memoryModelConfig = memorySpec?.modelConfig
                  ? `${agent.metadata.namespace}/${memorySpec.modelConfig}`
                  : "";
                const pt = decl?.promptTemplate;
                const srcRows: PromptSourceRow[] =
                  pt?.dataSources?.map((ds) => ({
                    id: crypto.randomUUID(),
                    name: ds.name || "",
                    alias: ds.alias || "",
                  })) ?? [newPromptSourceRow()];
                setState((prev) => ({
                  ...prev,
                  ...baseUpdates,
                  systemPrompt: decl?.systemMessage || "",
                  promptSourceRows: srcRows.length > 0 ? srcRows : [newPromptSourceRow()],
                  selectedTools: (decl?.tools && agentResponse.tools) ? agentResponse.tools : [],
                  selectedModel: agentResponse.modelConfigRef ? { ref: agentResponse.modelConfigRef, spec: { model: agentResponse.model || "", provider: "" } } : null,
                  inlineSkills: agent.spec?.declarative?.inlineSkills || [],
                  skillRefs: agent.spec?.skills?.refs || [],
                  stream: decl?.stream ?? false,
                  declarativeRuntime: decl?.runtime === "go" ? "go" : "python",
                  selectedMemoryModel: memoryModelConfig
                    ? { ref: memoryModelConfig, spec: { model: memorySpec?.modelConfig || "", provider: "" } }
                    : null,
                  memoryTtlDays: memorySpec?.ttlDays ? String(memorySpec.ttlDays) : "",
                  contextConfig: decl?.context,
                  serviceAccountName: decl?.deployment?.serviceAccountName || "",
                  byoImage: "",
                  byoCmd: "",
                  byoArgs: "",
                }));
              } else {
                setState((prev) => ({
                  ...prev,
                  ...baseUpdates,
                  systemPrompt: "",
                  selectedModel: null,
                  selectedTools: [],
                  selectedMemoryModel: null,
                  memoryTtlDays: "",
                  byoImage: agent.spec?.byo?.deployment?.image || "",
                  byoCmd: agent.spec?.byo?.deployment?.cmd || "",
                  byoArgs: (agent.spec?.byo?.deployment?.args || []).join(" "),
                  replicas: agent.spec?.byo?.deployment?.replicas !== undefined ? String(agent.spec?.byo?.deployment?.replicas) : "",
                  imagePullPolicy: agent.spec?.byo?.deployment?.imagePullPolicy || "",
                  imagePullSecrets: (agent.spec?.byo?.deployment?.imagePullSecrets || [])
                    .map((s: { name: string }) => s.name)
                    .concat((agent.spec?.byo?.deployment?.imagePullSecrets || []).length === 0 ? [""] : []),
                  envPairs: (agent.spec?.byo?.deployment?.env || [])
                    .map((e: EnvVar) =>
                      e?.valueFrom?.secretKeyRef
                        ? {
                            name: e.name || "",
                            isSecret: true,
                            secretName: e.valueFrom.secretKeyRef.name || "",
                            secretKey: e.valueFrom.secretKeyRef.key || "",
                            optional: e.valueFrom.secretKeyRef.optional,
                          }
                        : { name: e.name || "", value: e.value || "", isSecret: false },
                    )
                    .concat((agent.spec?.byo?.deployment?.env || []).length === 0
                      ? [{ name: "", value: "", isSecret: false }]
                      : []),
                  serviceAccountName: agent.spec?.byo?.deployment?.serviceAccountName || "",
                }));
              }
            } catch (extractError) {
              console.error("Error extracting assistant data:", extractError);
              toast.error("Failed to extract agent data");
            }
          } else {
            toast.error("Agent not found");
          }
        } catch (e) {
          console.error("Error fetching agent:", e);
          toast.error("Failed to load agent data");
        } finally {
          setState((prev) => ({ ...prev, isLoading: false }));
        }
      }
    };

    void fetchAgentData();
  }, [isEditMode, agentName, agentNamespace, getAgent, protectedAgentNames]);

  const validateForm = () => {
    const memoryEnabled = !!(state.selectedMemoryModel?.ref || state.memoryTtlDays);
    const formData = {
      name: state.name,
      namespace: state.namespace,
      description: state.description,
      type: state.agentType,
      systemPrompt: state.systemPrompt,
      promptSources: state.promptSourceRows.map(({ name, alias }) => ({ name, alias })),
      modelName: state.selectedModel?.ref || "",
      tools: state.selectedTools,
      byoImage: state.byoImage,
      memory: memoryEnabled
        ? {
          modelConfig: state.selectedMemoryModel?.ref || "",
          ttlDays: state.memoryTtlDays ? parseInt(state.memoryTtlDays, 10) : undefined,
        }
        : undefined,
      context: state.contextConfig,
      serviceAccountName: state.serviceAccountName,
    };

    const newErrors = validateAgentData(formData);

    if (state.agentType === "Declarative" && state.inlineSkills.length > 0) {
      for (const skill of state.inlineSkills) {
        if (!skill.name.trim() && !skill.description.trim() && !skill.content.trim()) continue; // empty entry, skip
        if (!skill.name.trim()) {
          newErrors.skills = "Each skill must have a name";
          break;
        }
        if (!skill.description.trim()) {
          newErrors.skills = `Skill "${skill.name}" must have a description`;
          break;
        }
        if (!skill.content.trim()) {
          newErrors.skills = `Skill "${skill.name}" must have content`;
          break;
        }
      }
      // Check for duplicate names
      if (!newErrors.skills) {
        const names = state.inlineSkills.filter(s => s.name.trim()).map(s => s.name.trim().toLowerCase());
        const dupName = names.find((n, i) => names.indexOf(n) !== i);
        if (dupName) {
          newErrors.skills = `Duplicate skill name: ${dupName}`;
        }
      }
    }

    // Validate CLI container refs
    if (state.agentType === "Declarative" && state.skillRefs.length > 0) {
      for (const ref of state.skillRefs) {
        if (!ref.trim()) continue;
        if (!isValidContainerImage(ref)) {
          newErrors.skills = `Invalid container image format: ${ref}`;
          break;
        }
      }
      if (!newErrors.skills) {
        const refs = state.skillRefs.filter(r => r.trim()).map(r => r.trim().toLowerCase());
        const dupRef = refs.find((r, i) => refs.indexOf(r) !== i);
        if (dupRef) {
          newErrors.skills = `Duplicate container image: ${dupRef}`;
        }
      }
    }

    setState((prev) => ({ ...prev, errors: newErrors }));
    const valid = Object.keys(newErrors).length === 0;
    if (!valid) {
      requestAnimationFrame(() => {
        focusFirstFormError(newErrors, { byoSectionsActive: showByoFields });
      });
    }
    return valid;
  };

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const validateField = (fieldName: keyof AgentFormValidationErrors, value: any) => {
    const formData: Partial<AgentFormData> = {};
    const memoryEnabled = !!(state.selectedMemoryModel?.ref || state.memoryTtlDays);

    switch (fieldName) {
      case "name":
        formData.name = value;
        break;
      case "namespace":
        formData.namespace = value;
        break;
      case "description":
        formData.description = value;
        break;
      case "type":
        formData.type = value;
        break;
      case "systemPrompt":
        formData.systemPrompt = value;
        break;
      case "model":
        formData.modelName = value;
        break;
      case "tools":
        formData.tools = value;
        break;
      case "memoryModel":
        if (memoryEnabled || value) {
          formData.memory = {
            modelConfig: value,
            ttlDays: state.memoryTtlDays ? parseInt(state.memoryTtlDays, 10) : undefined,
          };
        }
        break;
      case "memoryTtl":
        if (memoryEnabled || value) {
          formData.memory = {
            modelConfig: state.selectedMemoryModel?.ref || "",
            ttlDays: value ? parseInt(value, 10) : undefined,
          };
        }
        break;
      case "serviceAccountName":
        formData.serviceAccountName = value;
        break;
    }

    const fieldErrors = validateAgentData(formData);
    const valueForField = (fieldErrors as Record<string, string | undefined>)[fieldName as string];
    setState((prev) => ({
      ...prev,
      errors: {
        ...prev.errors,
        [fieldName]: valueForField,
      },
    }));
  };

  const handleSaveAgent = async () => {
    if (isViewMode) {
      return;
    }

    if (!validateForm()) {
      return;
    }

    try {
      setState((prev) => ({ ...prev, isSubmitting: true }));
      if (useDeclarativeAgentFields && !state.selectedModel) {
        throw new Error("Model is required for this agent type.");
      }

      const memoryEnabled = !!(state.selectedMemoryModel?.ref || state.memoryTtlDays);

      const agentData = {
        name: state.name,
        namespace: state.namespace,
        description: state.description,
        privateMode: state.privateMode,
        category: state.category || undefined,
        type: state.agentType,
        systemPrompt: state.systemPrompt,
        promptSources: state.promptSourceRows.map(({ name, alias }) => ({ name, alias })),
        modelName: state.selectedModel?.ref || "",
        stream: state.stream,
        tools: state.selectedTools,
        skillRefs: state.agentType === "Declarative" && state.skillRefs.length > 0
          ? state.skillRefs.filter(r => r.trim())
          : undefined,
        inlineSkills: state.agentType === "Declarative" && state.inlineSkills.length > 0
          ? state.inlineSkills.filter(s => s.name.trim() && s.description.trim() && s.content.trim())
          : undefined,
        memory: state.agentType === "Declarative" && memoryEnabled
          ? {
            modelConfig: state.selectedMemoryModel?.ref || "",
            ttlDays: state.memoryTtlDays ? parseInt(state.memoryTtlDays, 10) : undefined,
          }
          : undefined,
        context: useDeclarativeAgentFields ? state.contextConfig : undefined,
        declarativeRuntime: useDeclarativeAgentFields ? state.declarativeRuntime : undefined,
        byoImage: state.byoImage,
        byoCmd: state.byoCmd || undefined,
        byoArgs: state.byoArgs ? state.byoArgs.split(/\s+/).filter(Boolean) : undefined,
        replicas: state.replicas ? parseInt(state.replicas, 10) : undefined,
        imagePullPolicy: state.imagePullPolicy || undefined,
        imagePullSecrets: (state.imagePullSecrets || [])
          .filter((n) => n.trim())
          .map((n) => ({ name: n.trim() })),
        env: (state.envPairs || [])
          .map<EnvVar | null>((ev) => {
            const name = (ev.name || "").trim();
            if (!name) {
              return null;
            }
            if (ev.isSecret) {
              const secName = (ev.secretName || "").trim();
              const secKey = (ev.secretKey || "").trim();
              if (!secName || !secKey) {
                return null;
              }
              return {
                name,
                valueFrom: {
                  secretKeyRef: {
                    name: secName,
                    key: secKey,
                    optional: ev.optional,
                  },
                },
              } as EnvVar;
            }
            return { name, value: ev.value ?? "" } as EnvVar;
          })
          .filter((e): e is EnvVar => e !== null),
        serviceAccountName: state.serviceAccountName.trim() || undefined,
      };

      let result;

      if (isEditMode && agentName && agentNamespace) {
        result = await updateAgent(agentData);
      } else {
        result = await createNewAgent(agentData);
      }

      if (result.error) {
        throw new Error(result.error);
      }

      setFormDirty(false);
      router.push(`/agents`);
    } catch (e) {
      console.error(`Error ${isEditMode ? "updating" : "creating"} agent:`, e);
      const errorMessage =
        e instanceof Error ? e.message : `Failed to ${isEditMode ? "update" : "create"} agent. Please try again.`;
      toast.error(errorMessage);
      setState((prev) => ({ ...prev, isSubmitting: false }));
    }
  };

  const clearSkillsError = useCallback(() => {
    setState((prev) => ({ ...prev, errors: { ...prev.errors, skills: undefined } }));
  }, []);

  const renderPageContent = () => {
    if (error) {
      return <ErrorState message={error} />;
    }

    return (
      <div className="min-h-screen p-8">
        <div className="max-w-6xl mx-auto">
          <h1 className="text-2xl font-bold mb-8">{isViewMode ? "View Agent" : (isEditMode ? "Edit Agent" : "Create New Agent")}</h1>

          <fieldset disabled={isFormDisabled} className="space-y-6 min-w-0 border-0 p-0 m-0">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2 text-xl font-bold">
                  <KagentLogo className="h-5 w-5" />
                  Basic Information
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <label className="text-base mb-2 block font-bold">Agent Name</label>
                  <p className="text-xs mb-2 block text-muted-foreground">
                    This is the name of the agent that will be displayed in the UI and used to identify the agent.
                  </p>
                  <Input
                    value={state.name}
                    onChange={(e) => setState(prev => ({ ...prev, name: e.target.value }))}
                    onBlur={() => validateField('name', state.name)}
                    className={`${state.errors.name ? "border-red-500" : ""}`}
                    placeholder="Enter agent name..."
                    disabled={state.isSubmitting || state.isLoading || isEditMode}
                  />
                  {state.errors.name && <p className="text-red-500 text-sm mt-1">{state.errors.name}</p>}
                </div>

              <FieldRoot>
                <FieldLabel htmlFor="agent-field-namespace">Namespace</FieldLabel>
                <FieldHint>Must exist and match where ModelConfigs and tools are resolved.</FieldHint>
                <NamespaceCombobox
                  id="agent-field-namespace"
                  value={state.namespace}
                  onValueChange={(value) => {
                    setState((prev) => ({ ...prev, selectedModel: null, namespace: value }));
                    validateField("namespace", value);
                  }}
                  disabled={disabled || isEditMode}
                />
              </FieldRoot>

                <div>
                  <Label className="text-base mb-2 block font-bold">Agent Type</Label>
                  <p className="text-xs mb-2 block text-muted-foreground">
                    Declarative or Sandbox: model, tools, and prompts below. BYO: your own container image.
                  </p>
                  <Select
                    value={state.agentType}
                    onValueChange={(val) => {
                      const next = val as AgentType;
                      setState((prev) => ({
                        ...prev,
                        agentType: next,
                      }));
                      validateField("type", val);
                    }}
                    disabled={state.isSubmitting || state.isLoading}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select agent type" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Declarative">Declarative</SelectItem>
                      <SelectItem value="Sandbox" disabled={disableSandboxCreation}>{disableSandboxCreation ? "Sandbox (disabled)" : "Sandbox"}</SelectItem>
                      <SelectItem value="BYO" disabled={disableByoAgentCreation}>{disableByoAgentCreation ? "BYO (disabled)" : "BYO"}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                <div>
                  <label className="text-sm mb-2 block">Description</label>
                  <p className="text-xs mb-2 block text-muted-foreground">
                    This is a description of the agent. It&apos;s for your reference only and it&apos;s not going to be used by the agent.
                  </p>
                  <Textarea
                    value={state.description}
                    onChange={(e) => setState(prev => ({ ...prev, description: e.target.value }))}
                    onBlur={() => validateField('description', state.description)}
                    className={`min-h-[100px] ${state.errors.description ? "border-red-500" : ""}`}
                    placeholder="Describe your agent. This is for your reference only and it's not going to be used by the agent."
                    disabled={state.isSubmitting || state.isLoading}
                  />
                  {state.errors.description && <p className="text-red-500 text-sm mt-1">{state.errors.description}</p>}
                </div>
              
                <Label className="text-base mb-2 block font-bold">Agent Visibility</Label>
                <div className="flex items-center justify-between rounded-md border p-4">
                  <div className="space-y-1">
                    <Label htmlFor="private-mode-toggle" className="text-sm font-medium">Private or Public</Label>
                    <p className="text-xs text-muted-foreground">
                      Private agents are visible only to their owner. Public agents can be viewed by all users.
                    </p>
                  </div>
                  <Tabs
                    value={state.privateMode ? "private" : "public"}
                    onValueChange={(v) => {
                      if (v === "public") {
                        setShowPublicConfirm(true);
                      } else {
                        setState(prev => ({ ...prev, privateMode: true }));
                      }
                    }}
                  >
                    <TabsList>
                      <TabsTrigger value="public" disabled={state.isSubmitting || state.isLoading || !isEditMode}>Public</TabsTrigger>
                      <TabsTrigger value="private" disabled={state.isSubmitting || state.isLoading}>Private</TabsTrigger>
                    </TabsList>
                  </Tabs>
                </div>

                <AlertDialog open={showPublicConfirm} onOpenChange={setShowPublicConfirm}>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Make Agent Public?</AlertDialogTitle>
                      <AlertDialogDescription>
                        Public agents are visible to all users. Are you sure you want to change the visibility of this agent?
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={() => setState(prev => ({ ...prev, privateMode: false }))}>
                        Confirm
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>

                <div>
                  <label className="text-sm mb-2 block">Category (optional)</label>
                  <p className="text-xs mb-2 block text-muted-foreground">
                    Assign a category to group agents in the dashboard (e.g. velero, istio, prometheus).
                  </p>
                  <CategoryCombobox
                    value={state.category}
                    onValueChange={(value) => setState(prev => ({ ...prev, category: value }))}
                    disabled={state.isSubmitting || state.isLoading}
                  />
                </div>


              <FieldRoot>
                <FieldLabel htmlFor="agent-desc">Description (optional)</FieldLabel>
                <FieldHint>Internal note only; not sent to the model as instructions.</FieldHint>
                <Textarea
                  id="agent-desc"
                  name="description"
                  value={state.description}
                  onChange={(e) => setState((prev) => ({ ...prev, description: e.target.value }))}
                  onBlur={() => validateField("description", state.description)}
                  className={`min-h-[96px] ${state.errors.description ? "border-destructive" : ""}`}
                  placeholder="What this agent is for…"
                  autoComplete="off"
                  disabled={disabled}
                  aria-invalid={!!state.errors.description}
                />
                <FieldError>{state.errors.description}</FieldError>
              </FieldRoot>
              </CardContent>
            </Card>

            {useDeclarativeAgentFields && (
              <FormSection
                title="Model & behavior"
                description="Instructions, main model, streaming, and optional pod service account for this declarative or sandbox agent."
              >
                <SystemPromptSection
                  value={state.systemPrompt}
                  onChange={(e) => setState((prev) => ({ ...prev, systemPrompt: e.target.value }))}
                  onBlur={() => validateField("systemPrompt", state.systemPrompt)}
                  error={state.errors.systemPrompt}
                  disabled={disabled}
                  mentionNamespace={state.namespace}
                  onPickInclude={(pick) => ensureConfigMapSource(pick.configMapName)}
                  includeSourceIdForConfigMap={includeSourceIdForConfigMap}
                />

                <ModelSelectionSection
                  allModels={models}
                  selectedModel={state.selectedModel}
                  setSelectedModel={(model) => {
                    setState((prev) => ({ ...prev, selectedModel: model as ModelConfig | null }));
                  }}
                  error={state.errors.model}
                  isSubmitting={disabled}
                  onChange={(modelRef) => validateField("model", modelRef)}
                  agentNamespace={state.namespace}
                />

                <div className="flex gap-3 rounded-md border border-border/60 bg-muted/20 p-3">
                  <div className="flex h-5 shrink-0 items-center self-start">
                    <Checkbox
                      id="stream-toggle"
                      checked={state.stream}
                      onCheckedChange={(checked) => setState((prev) => ({ ...prev, stream: !!checked }))}
                      disabled={disabled}
                    />
                  </div>
                  <label htmlFor="stream-toggle" className="cursor-pointer select-none">
                    <span className="text-sm font-medium leading-tight">Enable Streaming</span>
                    <span className="mt-0.5 block text-xs text-muted-foreground">
                      Stream the model&apos;s response token-by-token instead of waiting for the full reply.
                    </span>
                  </label>
                </div>
              </FormSection>
            )}
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Layers className="h-5 w-5 text-yellow-500" />
                      Tools & Agents
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <ToolsSection
                      selectedTools={state.selectedTools}
                      setSelectedTools={(tools) => setState(prev => ({ ...prev, selectedTools: tools }))}
                      isSubmitting={state.isSubmitting || state.isLoading}
                      onBlur={() => validateField('tools', state.selectedTools)}
                      currentAgentName={state.name}
                      currentAgentNamespace={state.namespace}
                    />
                  </CardContent>
                </Card>

                <FormSection
                  title="Long-term memory"
                  description="Optional: embed and recall information across sessions using a dedicated model config for embeddings."
                >
                  <MemorySection
                    allModels={models}
                    selectedModel={state.selectedMemoryModel}
                    setSelectedModel={(model) => {
                      setState((prev) => ({ ...prev, selectedMemoryModel: model as ModelConfig | null }));
                      validateField("memoryModel", (model as ModelConfig | null)?.ref || "");
                    }}
                    agentNamespace={state.namespace}
                    ttlDays={state.memoryTtlDays}
                    onTtlChange={(value) => setState((prev) => ({ ...prev, memoryTtlDays: value }))}
                    onTtlBlur={() => validateField("memoryTtl", state.memoryTtlDays)}
                    modelError={state.errors.memoryModel}
                    ttlError={state.errors.memoryTtl}
                    isSubmitting={disabled}
                  />
                </FormSection>

                <FormSection
                  title="Context"
                  description="Compaction and summarization to keep long runs within model limits. Leave off for the default context behavior."
                >
                  <ContextSection
                    context={state.contextConfig}
                    onChange={(ctx) => setState((prev) => ({ ...prev, contextConfig: ctx }))}
                    isSubmitting={disabled}
                  />
                </FormSection>

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Layers className="h-5 w-5 text-blue-500" />
                      Skills
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <p className="text-xs text-muted-foreground">
                        Skills are prompt instructions that guide the agent. They can reference CLI tools mounted from containers at <code>/skills/&lt;container-name&gt;/scripts/...</code>
                      </p>
                      <div className="space-y-3">
                        {state.inlineSkills.map((skill, idx) => (
                            <div key={idx} className="border rounded-md p-4 space-y-3">
                              <div className="flex gap-2 items-start">
                                <div className="flex-1 space-y-3">
                                  <div className="flex gap-2">
                                    <Input
                                      placeholder="Skill name (e.g., data-analysis)"
                                      value={skill.name}
                                      onChange={(e) => {
                                        const copy = [...state.inlineSkills];
                                        copy[idx] = { ...copy[idx], name: e.target.value.toLowerCase().replace(/[^a-z0-9.\-]/g, "-") };
                                        setState(prev => ({ ...prev, inlineSkills: copy }));
                                      }}
                                      disabled={isFormDisabled}
                                      className="font-mono text-sm flex-1"
                                    />
                                    <Input
                                      placeholder="Brief description (required)"
                                      value={skill.description}
                                      onChange={(e) => {
                                        const copy = [...state.inlineSkills];
                                        copy[idx] = { ...copy[idx], description: e.target.value };
                                        setState(prev => ({ ...prev, inlineSkills: copy }));
                                      }}
                                      disabled={isFormDisabled}
                                      className="flex-1"
                                    />
                                  </div>
                                  <Textarea
                                    placeholder="Skill instructions (Markdown)"
                                    value={skill.content}
                                    onChange={(e) => {
                                      const copy = [...state.inlineSkills];
                                      copy[idx] = { ...copy[idx], content: e.target.value };
                                      setState(prev => ({ ...prev, inlineSkills: copy }));
                                    }}
                                    disabled={isFormDisabled}
                                    className="min-h-[100px] font-mono text-sm"
                                  />
                                </div>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  onClick={() => setState(prev => ({
                                    ...prev,
                                    inlineSkills: prev.inlineSkills.filter((_, i) => i !== idx)
                                  }))}
                                  disabled={isFormDisabled}
                                  title="Remove skill"
                                >
                                  <Trash2 className="h-4 w-4 text-red-500" />
                                </Button>
                              </div>
                            </div>
                        ))}
                        <Button
                          variant="outline"
                          onClick={() => {
                            if (state.inlineSkills.length < 20) {
                              setState(prev => ({
                                ...prev,
                                inlineSkills: [...prev.inlineSkills, { name: "", description: "", content: "" }]
                              }));
                            }
                          }}
                          disabled={isFormDisabled || state.inlineSkills.length >= 20}
                        >
                          <PlusCircle className="h-4 w-4 mr-2" /> Add Skill
                        </Button>
                      </div>
                      {state.errors.skills && (
                        <p className="text-red-500 text-sm mt-2">{state.errors.skills}</p>
                      )}
                    </div>
                  </CardContent>
                </Card>

                {(!disableCliContainers || skillCliPresets.length > 0) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Package className="h-5 w-5 text-orange-500" />
                      CLI Containers
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-4">
                      <p className="text-xs text-muted-foreground">
                        Container images are extracted to <code>/skills/&lt;name&gt;/</code> and available to all skills.
                      </p>
                      {skillCliPresets.length > 0 && (
                        <div>
                          <Label className="text-xs text-muted-foreground mb-1 block">Quick add from presets:</Label>
                          <div className="flex flex-wrap gap-2">
                            {skillCliPresets.map((preset) => {
                              const alreadyAdded = state.skillRefs.some(r => r === preset.ref);
                              return (
                                <Button
                                  key={preset.ref}
                                  variant="outline"
                                  size="sm"
                                  title={preset.description || preset.ref}
                                  disabled={isFormDisabled || alreadyAdded}
                                  onClick={() => {
                                    setState(prev => ({
                                      ...prev,
                                      skillRefs: [...prev.skillRefs, preset.ref]
                                    }));
                                  }}
                                >
                                  <PlusCircle className="h-3 w-3 mr-1" /> {preset.label}
                                </Button>
                              );
                            })}
                          </div>
                        </div>
                      )}
                      {disableCliContainers && state.skillRefs.length > 0 && (
                        <div className="space-y-2">
                          {state.skillRefs.map((ref, idx) => (
                            <div key={idx} className="flex gap-2 items-center">
                              <Input
                                value={ref}
                                disabled
                                className="font-mono text-sm"
                              />
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => setState(prev => ({
                                  ...prev,
                                  skillRefs: prev.skillRefs.filter((_, i) => i !== idx)
                                }))}
                                disabled={isFormDisabled}
                                title="Remove container"
                              >
                                <Trash2 className="h-4 w-4 text-red-500" />
                              </Button>
                            </div>
                          ))}
                        </div>
                      )}
                      {!disableCliContainers && (
                      <div className="space-y-2">
                        {state.skillRefs.map((ref, idx) => {
                          const refInvalid = ref.trim() && !isValidContainerImage(ref);
                          return (
                            <div key={idx} className="flex gap-2 items-center">
                              <Input
                                placeholder="Container image (e.g., ghcr.io/org/skill:v1.0)"
                                value={ref}
                                onChange={(e) => {
                                  const copy = [...state.skillRefs];
                                  copy[idx] = e.target.value;
                                  setState(prev => ({ ...prev, skillRefs: copy }));
                                }}
                                disabled={isFormDisabled}
                                className={refInvalid ? "border-red-500 font-mono text-sm" : "font-mono text-sm"}
                              />
                              <Button
                                variant="ghost"
                                size="icon"
                                onClick={() => setState(prev => ({
                                  ...prev,
                                  skillRefs: prev.skillRefs.filter((_, i) => i !== idx)
                                }))}
                                disabled={isFormDisabled}
                                title="Remove container"
                              >
                                <Trash2 className="h-4 w-4 text-red-500" />
                              </Button>
                            </div>
                          );
                        })}
                        <Button
                          variant="outline"
                          onClick={() => setState(prev => ({ ...prev, skillRefs: [...prev.skillRefs, ""] }))}
                          disabled={isFormDisabled}
                        >
                          <PlusCircle className="h-4 w-4 mr-2" /> Add Container
                        </Button>
                      </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
                )}
            {!isViewMode && (
              <div className="flex justify-end">
                <Button className="bg-violet-500 hover:bg-violet-600" onClick={handleSaveAgent} disabled={state.isSubmitting || state.isLoading}>
                  {state.isSubmitting ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      {isEditMode ? "Updating..." : "Creating..."}
                    </>
                  ) : isEditMode ? (
                    "Update Agent"
                  ) : (
                    "Create Agent"
                  )}
                </Button>
              </div>
            )}
          </fieldset>
        </div>
      </div>
    );
  };

  return (
    <>
      {(loading || state.isLoading) && <LoadingState />}
      {renderPageContent()}
    </>
  );
}

export default function AgentPage() {
  const searchParams = useSearchParams();
  const isEditMode = searchParams.get("edit") === "true";
  const isViewMode = searchParams.get("readonly") === "true";
  const agentName = searchParams.get("name");
  const agentNamespace = searchParams.get("namespace");
  const formKey = isEditMode ? `edit-${agentName}-${agentNamespace}` : "create";

  return (
    <Suspense fallback={<LoadingState />}>
      <AgentPageContent key={formKey} isEditMode={isEditMode} isViewMode={isViewMode} agentName={agentName} agentNamespace={agentNamespace} />
    </Suspense>
  );
}
