"use client";

import { useEffect, useState } from "react";
import { ChevronRight, Edit, ShieldAlert } from "lucide-react";
import type { AgentResponse, Tool, ToolsResponse } from "@/types";
import { SidebarHeader, Sidebar, SidebarContent, SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuItem, SidebarMenuButton } from "@/components/ui/sidebar";
import { ScrollArea } from "@/components/ui/scroll-area";
import { LoadingState } from "@/components/LoadingState";
import { isAgentTool, isMcpTool, getToolDescription, getToolIdentifier, getToolDisplayName } from "@/lib/toolUtils";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import Link from "next/link";
import { getAgents } from "@/app/actions/agents";
import { k8sRefUtils } from "@/lib/k8sUtils";
import { useUserStore } from "@/lib/userStore";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Badge } from "@/components/ui/badge";

interface AgentDetailsSidebarProps {
  selectedAgentName: string;
  currentAgent: AgentResponse;
  allTools: ToolsResponse[];
}

export function AgentDetailsSidebar({ selectedAgentName, currentAgent, allTools }: AgentDetailsSidebarProps) {
  const [toolDescriptions, setToolDescriptions] = useState<Record<string, string>>({});
  const [expandedTools, setExpandedTools] = useState<Record<string, boolean>>({});
  const [availableAgents, setAvailableAgents] = useState<AgentResponse[]>([]);
  const currentUserId = useUserStore((state) => state.userId);

  const selectedTeam = currentAgent;
  const ownerId = currentAgent.user_id || currentAgent.agent.metadata.annotations?.["kagent.dev/user-id"] || "";
  const isOwner = ownerId === currentUserId;

  // Fetch agents for looking up agent tool descriptions
  useEffect(() => {
    const fetchAgents = async () => {
      try {
        const response = await getAgents();
        if (response.data) {
          setAvailableAgents(response.data);

        } else if (response.error) {
          console.error("AgentDetailsSidebar: Error fetching agents:", response.error);
        }
      } catch (error) {
        console.error("AgentDetailsSidebar: Failed to fetch agents:", error);
      }
    };

    fetchAgents();
  }, []);



  const RenderToolCollapsibleItem = ({
    itemKey,
    displayName,
    providerTooltip,
    description,
    requiresApproval,
    isExpanded,
    onToggleExpansion,
  }: {
    itemKey: string;
    displayName: string;
    providerTooltip: string;
    description: string;
    requiresApproval?: boolean;
    isExpanded: boolean;
    onToggleExpansion: () => void;
  }) => {
    return (
      <Collapsible
        key={itemKey}
        open={isExpanded}
        onOpenChange={onToggleExpansion}
        className="group/collapsible"
      >
        <SidebarMenuItem>
          <CollapsibleTrigger asChild>
            <SidebarMenuButton tooltip={providerTooltip} className="w-full">
              <div className="flex items-center justify-between w-full">
                <span className="truncate max-w-[200px]">{displayName}</span>
                <div className="flex items-center gap-1">
                  {requiresApproval && (
                    <ShieldAlert className="h-3.5 w-3.5 text-amber-500 shrink-0" />
                  )}
                  <ChevronRight
                    className={cn(
                      "h-4 w-4 transition-transform duration-200",
                      isExpanded && "rotate-90"
                    )}
                  />
                </div>
              </div>
            </SidebarMenuButton>
          </CollapsibleTrigger>
          <CollapsibleContent className="px-2 py-1">
            <div className="rounded-md bg-muted/50 p-2">
              <p className="text-sm text-muted-foreground">{description}</p>
              {requiresApproval && (
                <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">Requires approval before execution</p>
              )}
            </div>
          </CollapsibleContent>
        </SidebarMenuItem>
      </Collapsible>
    );
  };

  useEffect(() => {
    const processToolDescriptions = () => {
      setToolDescriptions({});

      if (!selectedTeam || !allTools) return;

      const descriptions: Record<string, string> = {};
      const toolRefs = selectedTeam.tools;

      if (toolRefs && Array.isArray(toolRefs)) {
        toolRefs.forEach((tool) => {
          if (isMcpTool(tool)) {
            const mcpTool = tool as Tool;
            // For MCP tools, each tool name gets its own description
            const baseToolIdentifier = getToolIdentifier(mcpTool);
            mcpTool.mcpServer?.toolNames.forEach((mcpToolName) => {
              const subToolIdentifier = `${baseToolIdentifier}::${mcpToolName}`;
              
              // Find the tool in allTools by matching server ref and tool name
              const toolFromDB = allTools.find(server => {
                const { name } = k8sRefUtils.fromRef(server.server_name);
                return name === mcpTool.mcpServer?.name && server.id === mcpToolName;
              });

              if (toolFromDB) {
                descriptions[subToolIdentifier] = toolFromDB.description;
              } else {
                descriptions[subToolIdentifier] = "No description available";
              }
            });
          } else {
            // Handle Agent tools or regular tools using getToolDescription
            const toolIdentifier = getToolIdentifier(tool);
            descriptions[toolIdentifier] = getToolDescription(tool, allTools);
          }
        });
      }
      
      setToolDescriptions(descriptions);
    };

    processToolDescriptions();
  }, [selectedTeam, allTools, availableAgents]);

  const toggleToolExpansion = (toolIdentifier: string) => {
    setExpandedTools(prev => ({
      ...prev,
      [toolIdentifier]: !prev[toolIdentifier]
    }));
  };

  if (!selectedTeam) {
    return <LoadingState />;
  }

  const renderAgentTools = (tools: Tool[] = []) => {
    if (!tools || tools.length === 0) {
      return (
        <SidebarMenu>
          <div className="text-sm italic">No tools/agents available</div>
        </SidebarMenu>
      );
    }

    const agentNamespace = currentAgent.agent.metadata.namespace || "";

    return (
      <SidebarMenu>
        {tools.flatMap((tool) => {
          const baseToolIdentifier = getToolIdentifier(tool);

          if (tool.mcpServer && tool.mcpServer?.toolNames && tool.mcpServer.toolNames.length > 0) {
            const mcpProvider = tool.mcpServer.name || "mcp_server";
            const mcpProviderParts = mcpProvider.split(".");
            const mcpProviderNameTooltip = mcpProviderParts[mcpProviderParts.length - 1];
            const serverDisplayName = `${tool.mcpServer.namespace || agentNamespace}/${tool.mcpServer.name || ""}`;
            const approvalSet = new Set(tool.mcpServer.requireApproval || []);

            return tool.mcpServer.toolNames.map((mcpToolName) => {
              const subToolIdentifier = `${baseToolIdentifier}::${mcpToolName}`;
              const description = toolDescriptions[subToolIdentifier] || "Description loading or unavailable";
              const isExpanded = expandedTools[subToolIdentifier] || false;
              const displayName = `${mcpToolName} (${serverDisplayName})`;

              return (
                <RenderToolCollapsibleItem
                  key={subToolIdentifier}
                  itemKey={subToolIdentifier}
                  displayName={displayName}
                  providerTooltip={mcpProviderNameTooltip}
                  description={description}
                  requiresApproval={approvalSet.has(mcpToolName)}
                  isExpanded={isExpanded}
                  onToggleExpansion={() => toggleToolExpansion(subToolIdentifier)}
                />
              );
            });
          } else {
            const toolIdentifier = baseToolIdentifier;
            const provider = isAgentTool(tool) ? (tool.agent?.name || "unknown") : (tool.mcpServer?.name || "unknown");
            const displayName = getToolDisplayName(tool, agentNamespace);
            const description = toolDescriptions[toolIdentifier] || "Description loading or unavailable";
            const isExpanded = expandedTools[toolIdentifier] || false;

            const providerParts = provider.split(".");
            const providerNameTooltip = providerParts[providerParts.length - 1];

            return [(
              <RenderToolCollapsibleItem
                key={toolIdentifier}
                itemKey={toolIdentifier}
                displayName={displayName}
                providerTooltip={providerNameTooltip}
                description={description}
                isExpanded={isExpanded}
                onToggleExpansion={() => toggleToolExpansion(toolIdentifier)}
              />
            )];
          }
        })}
      </SidebarMenu>
    );
  };

  // Declarative agents (including SandboxAgent with declarative spec) share model-backed config.
  const isDeclarativeLikeAgent = selectedTeam?.agent.spec.type === "Declarative";

  return (
    <>
      <Sidebar
        side={"right"}
        collapsible="offcanvas"
        className="md:top-[101px] md:h-[calc(100svh-101px)]"
      >
        <SidebarHeader>Agent Details</SidebarHeader>
        <SidebarContent>
          <ScrollArea>
            <SidebarGroup>
              <div className="flex items-center justify-between px-2 mb-1">
                <SidebarGroupLabel className="font-bold mb-0 p-0">
                  {selectedTeam?.agent.metadata.namespace}/{selectedTeam?.agent.metadata.name} {selectedTeam?.model && `(${selectedTeam?.model})`}
                </SidebarGroupLabel>
                {isOwner && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    asChild
                    aria-label={`Edit agent ${selectedTeam?.agent.metadata.namespace}/${selectedTeam?.agent.metadata.name}`}
                  >
                    <Link href={`/agents/new?edit=true&name=${selectedAgentName}&namespace=${currentAgent.agent.metadata.namespace}`}>
                      <Edit className="h-3.5 w-3.5" />
                    </Link>
                  </Button>
                )}
              </div>
              <p className="text-sm flex px-2 text-muted-foreground">{selectedTeam?.agent.spec.description}</p>
            </SidebarGroup>
            {isDeclarativeLikeAgent && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <SidebarGroupLabel>Tools & Agents</SidebarGroupLabel>
                {selectedTeam && renderAgentTools(selectedTeam.tools)}
              </SidebarGroup>
            )}

            {isDeclarativeLikeAgent && selectedTeam?.agent.spec?.declarative?.inlineSkills && selectedTeam.agent.spec.declarative.inlineSkills.length > 0 && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <div className="flex items-center justify-between px-2 mb-2">
                  <SidebarGroupLabel className="mb-0">Skills</SidebarGroupLabel>
                  <Badge variant="secondary" className="h-5">
                    {selectedTeam.agent.spec.declarative.inlineSkills.length}
                  </Badge>
                </div>
                <SidebarMenu>
                  <TooltipProvider>
                    {selectedTeam.agent.spec.declarative.inlineSkills.map((skill, index) => (
                      <SidebarMenuItem key={index}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <SidebarMenuButton className="w-full h-auto py-2">
                              <div className="flex flex-col items-start w-full min-w-0 gap-0.5">
                                <span className="truncate text-sm font-medium leading-tight">{skill.name}</span>
                                <span className="truncate w-full text-xs text-muted-foreground leading-tight">
                                  {skill.description}
                                </span>
                              </div>
                            </SidebarMenuButton>
                          </TooltipTrigger>
                          <TooltipContent side="left">
                            <div className="max-w-xs">
                              <p className="font-medium">{skill.name}</p>
                              {skill.description && <p className="text-xs mt-1">{skill.description}</p>}
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      </SidebarMenuItem>
                    ))}
                  </TooltipProvider>
                </SidebarMenu>
              </SidebarGroup>
            )}

            {isDeclarativeLikeAgent && selectedTeam?.agent.spec?.skills?.refs && selectedTeam.agent.spec.skills.refs.length > 0 && (
              <SidebarGroup className="group-data-[collapsible=icon]:hidden">
                <div className="flex items-center justify-between px-2 mb-2">
                  <SidebarGroupLabel className="mb-0">CLI Containers</SidebarGroupLabel>
                  <Badge variant="secondary" className="h-5">
                    {selectedTeam.agent.spec.skills.refs.length}
                  </Badge>
                </div>
                <SidebarMenu>
                  <TooltipProvider>
                    {selectedTeam.agent.spec.skills.refs.map((ref, index) => {
                      const parts = ref.split("/");
                      const shortName = parts[parts.length - 1]?.split(":")[0] || ref;
                      return (
                        <SidebarMenuItem key={index}>
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <SidebarMenuButton className="w-full h-auto py-2">
                                <div className="flex flex-col items-start w-full min-w-0 gap-0.5">
                                  <span className="truncate text-sm font-medium leading-tight">{shortName}</span>
                                  <span className="truncate w-full text-xs text-muted-foreground/70 leading-tight font-mono">
                                    {ref}
                                  </span>
                                </div>
                              </SidebarMenuButton>
                            </TooltipTrigger>
                            <TooltipContent side="left">
                              <p className="break-all font-mono text-xs">{ref}</p>
                            </TooltipContent>
                          </Tooltip>
                        </SidebarMenuItem>
                      );
                    })}
                  </TooltipProvider>
                </SidebarMenu>
              </SidebarGroup>
            )}

          </ScrollArea>
        </SidebarContent>
      </Sidebar>
    </>
  );
}
