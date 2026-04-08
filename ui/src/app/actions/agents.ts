"use server";

import { AgentSpec, BaseResponse } from "@/types";
import { Agent, AgentResponse, Tool } from "@/types";
import { revalidatePath } from "next/cache";
import { fetchApi, createErrorResponse, getCurrentUserId } from "./utils";
import { AgentFormData } from "@/components/AgentsProvider";
import { isMcpTool } from "@/lib/toolUtils";
import { k8sRefUtils } from "@/lib/k8sUtils";
import { LABEL_CATEGORY, LABEL_TOOL_TYPE, LABEL_ROLE } from "@/lib/constants";

const PRIVATE_MODE_ANNOTATION = "kagent.dev/private-mode";
const USER_ID_ANNOTATION = "kagent.dev/user-id";

function getAgentPrivateMode(agentResponse: AgentResponse): boolean {
  if (typeof agentResponse.private_mode === "boolean") {
    return agentResponse.private_mode;
  }

  const annotationValue = agentResponse.agent.metadata.annotations?.[PRIVATE_MODE_ANNOTATION];
  if (annotationValue === "true") {
    return true;
  }
  if (annotationValue === "false") {
    return false;
  }

  // No explicit visibility set. Agents created via kubectl/helm typically have
  // neither private_mode nor user_id — treat them as public so they are visible
  // to everyone. Agents created through the UI always have both annotations.
  const hasOwner = !!(agentResponse.user_id || agentResponse.agent.metadata.annotations?.[USER_ID_ANNOTATION]);
  if (!hasOwner) {
    return false;
  }

  // Has an owner but no explicit visibility — default to private.
  return true;
}

function getAgentOwnerId(agentResponse: AgentResponse): string {
  return agentResponse.user_id || agentResponse.agent.metadata.annotations?.[USER_ID_ANNOTATION] || "";
}

/**
 * Converts AgentFormData to Agent format
 * @param agentFormData The form data to convert
 * @returns An Agent object
 */
function fromAgentFormDataToAgent(agentFormData: AgentFormData): Agent {
  const modelConfigName = agentFormData.modelName?.includes("/")
    ? agentFormData.modelName.split("/").pop() || ""
    : agentFormData.modelName;

  const type = agentFormData.type || "Declarative";
  const agentNamespace = agentFormData.namespace || "";

  const convertTools = (tools: Tool[]) =>
    tools.map((tool) => {
      if (isMcpTool(tool)) {
        const mcpServer = tool.mcpServer;
        if (!mcpServer) {
          throw new Error("MCP server not found");
        }
        
        let name = mcpServer.name;
        let namespace: string | undefined = mcpServer.namespace;
        
        if (k8sRefUtils.isValidRef(mcpServer.name)) {
          const parsed = k8sRefUtils.fromRef(mcpServer.name);
          name = parsed.name;
          // Ignore namespace on the name ref if one is set - using namespace/name format is legacy behavior
        }
        
        // If no namespace is set, default to the agent's namespace
        if (!namespace) {
          namespace = agentNamespace;
        }

        return {
          type: "McpServer",
          mcpServer: {
            name,
            namespace,
            kind: mcpServer.kind,
            apiGroup: mcpServer.apiGroup,
            toolNames: mcpServer.toolNames,
          },
        } as Tool;
      }

      if (tool.type === "Agent") {
        const agent = tool.agent;
        if (!agent) {
          throw new Error("Agent not found");
        }

        let name = agent.name;
        let namespace: string | undefined = agent.namespace;
        
        if (k8sRefUtils.isValidRef(name)) {
          const parsed = k8sRefUtils.fromRef(name);
          name = parsed.name;
          // Ignore namespace on the name ref if one is set - using namespace/name format is legacy behavior
        }
        
        // If no namespace is set, default to the agent's namespace
        if (!namespace) {
          namespace = agentNamespace;
        }
        
        return {
          type: "Agent",
          agent: {
            name,
            namespace,
            kind: agent.kind || "Agent",
            apiGroup: agent.apiGroup || "kagent.dev",
          },
        } as Tool;
      }

      console.warn("Unknown tool type:", tool);
      return tool as Tool;
    });

  // Auto-detect tool type from the agent's tools and skills
  const hasMcp = (agentFormData.tools || []).some((t) => isMcpTool(t));
  const hasSkill = !!(agentFormData.skillRefs && agentFormData.skillRefs.length > 0);
  let computedToolType = "";
  if (hasMcp && hasSkill) {
    computedToolType = "mcp+skill";
  } else if (hasMcp) {
    computedToolType = "mcp";
  } else if (hasSkill) {
    computedToolType = "skill";
  }

  const labels: Record<string, string> = {};
  if (agentFormData.category) {
    labels[LABEL_CATEGORY] = agentFormData.category;
  }
  if (computedToolType) {
    labels[LABEL_TOOL_TYPE] = computedToolType;
  }
  if (agentFormData.role) {
    labels[LABEL_ROLE] = agentFormData.role;
  }

  const base: Partial<Agent> = {
    metadata: {
      name: agentFormData.name,
      namespace: agentFormData.namespace || "",
      annotations: {
        [PRIVATE_MODE_ANNOTATION]: String(agentFormData.privateMode ?? true),
      },
      ...(Object.keys(labels).length > 0 && { labels }),
    },
    spec: {
      type,
      description: agentFormData.description,
    } as AgentSpec,
  };

  if (type === "Declarative") {
    base.spec!.declarative = {
      systemMessage: agentFormData.systemPrompt || "",
      modelConfig: modelConfigName || "",
      stream: agentFormData.stream ?? true,
      tools: convertTools(agentFormData.tools || []),
    };

    if (agentFormData.skillRefs && agentFormData.skillRefs.length > 0) {
      base.spec!.skills = {
        refs: agentFormData.skillRefs,
      };
    }

    if (agentFormData.memory?.modelConfig) {
      const memoryModel = agentFormData.memory.modelConfig;
      const memoryModelName = k8sRefUtils.isValidRef(memoryModel)
        ? k8sRefUtils.fromRef(memoryModel).name
        : memoryModel;
      base.spec!.memory = {
        modelConfig: memoryModelName,
        ttlDays: agentFormData.memory.ttlDays,
      };
    }

    if (agentFormData.context) {
      base.spec!.declarative!.context = agentFormData.context;
    }

    const trimmedSA = agentFormData.serviceAccountName?.trim();
    if (trimmedSA) {
      base.spec!.declarative!.deployment = {
        ...base.spec!.declarative!.deployment,
        serviceAccountName: trimmedSA,
      };
    }
  } else if (type === "BYO") {
    base.spec!.byo = {
      deployment: {
        image: agentFormData.byoImage || "",
        cmd: agentFormData.byoCmd,
        args: agentFormData.byoArgs,
        replicas: agentFormData.replicas,
        imagePullSecrets: agentFormData.imagePullSecrets,
        volumes: agentFormData.volumes,
        volumeMounts: agentFormData.volumeMounts,
        labels: agentFormData.labels,
        annotations: agentFormData.annotations,
        env: agentFormData.env,
        imagePullPolicy: agentFormData.imagePullPolicy,
        serviceAccountName: agentFormData.serviceAccountName,
      },
    };
  }

  return base as Agent;
}

export async function getAgent(agentName: string, namespace: string): Promise<BaseResponse<AgentResponse>> {
  try {
    const agentData = await fetchApi<BaseResponse<AgentResponse>>(`/agents/${namespace}/${agentName}`);
    return { message: "Successfully fetched agent", data: agentData.data };
  } catch (error) {
    return createErrorResponse<AgentResponse>(error, "Error getting agent");
  }
}

/**
 * Deletes a agent
 * @param agentName The agent name
 * @param namespace The agent namespace
 * @returns A promise with the delete result
 */
export async function deleteAgent(agentName: string, namespace: string): Promise<BaseResponse<void>> {
  try {
    await fetchApi(`/agents/${namespace}/${agentName}`, {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
      },
    });

    revalidatePath("/");
    return { message: "Successfully deleted agent" };
  } catch (error) {
    return createErrorResponse<void>(error, "Error deleting agent");
  }
}

/**
 * Creates or updates an agent
 * @param agentConfig The agent configuration
 * @param update Whether to update an existing agent
 * @returns A promise with the created/updated agent
 */
export async function createAgent(agentConfig: AgentFormData, update: boolean = false): Promise<BaseResponse<Agent>> {
  try {
    // Only get the name of the model, not the full ref
    if (agentConfig.modelName) {
      if (k8sRefUtils.isValidRef(agentConfig.modelName)) {
        agentConfig.modelName = k8sRefUtils.fromRef(agentConfig.modelName).name;
      }
    }

    const agentPayload = fromAgentFormDataToAgent(agentConfig);

    // Ensure user-id annotation is set so the agent is visible to its creator
    const userId = await getCurrentUserId();
    if (agentPayload.metadata?.annotations) {
      agentPayload.metadata.annotations[USER_ID_ANNOTATION] = userId;
    }

    const response = await fetchApi<BaseResponse<Agent>>(`/agents`, {
      method: update ? "PUT" : "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(agentPayload),
    });

    if (!response) {
      throw new Error("Failed to create agent");
    }

    const agentRef = k8sRefUtils.toRef(
      response.data!.metadata.namespace || "",
      response.data!.metadata.name,
    )

    revalidatePath("/agents");
    revalidatePath(`/agents/${agentRef}/chat`);
    return { message: "Successfully created agent", data: response.data };
  } catch (error) {
    return createErrorResponse<Agent>(error, "Error creating agent");
  }
}

/**
 * Gets all agents
 * @returns A promise with all agents
 */
export async function getAgents(): Promise<BaseResponse<AgentResponse[]>> {
  try {
    const currentUserId = await getCurrentUserId();
    const { data } = await fetchApi<BaseResponse<AgentResponse[]>>(`/agents`);

    const visibleAgents = data?.filter((agentResponse) => {
      const isOwner = getAgentOwnerId(agentResponse) === currentUserId;
      if (isOwner) {
        return true;
      }

      return getAgentPrivateMode(agentResponse) === false;
    });

    const sortedData = visibleAgents?.sort((a, b) => {
      const aRef = k8sRefUtils.toRef(a.agent.metadata.namespace || "", a.agent.metadata.name);
      const bRef = k8sRefUtils.toRef(b.agent.metadata.namespace || "", b.agent.metadata.name);
      return aRef.localeCompare(bRef);
    });

    return { message: "Successfully fetched agents", data: sortedData };
  } catch (error) {
    return createErrorResponse<AgentResponse[]>(error, "Error getting agents");
  }
}
