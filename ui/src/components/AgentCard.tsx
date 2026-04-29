"use client";

import { useState } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { AgentResponse } from "@/types";
import { DeleteButton } from "@/components/DeleteAgentButton";
import { MemoriesDialog } from "@/components/MemoriesDialog";
import KagentLogo from "@/components/kagent-logo";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Brain, Eye, MoreHorizontal, Pencil, Trash2, Shield, User } from "lucide-react";
import { cn } from "@/lib/utils";
import { useAppConfig, isEffectivelyProtectedCheck } from "@/lib/configStore";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useUserStore } from "@/lib/userStore";
import { LABEL_TOOL_TYPE, LABEL_CATEGORY } from "@/lib/constants";

interface AgentCardProps {
  agentResponse: AgentResponse;
}

export function AgentCard({ agentResponse }: AgentCardProps) {
  const { agent, model, modelProvider, deploymentReady, accepted, private_mode, user_id } = agentResponse;
  const router = useRouter();
  const currentUserId = useUserStore((state) => state.userId);
  const { protectedAgentNames } = useAppConfig();
  const [memoriesOpen, setMemoriesOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const isBYO = agent.spec?.type === "BYO";
  const byoImage = isBYO ? agent.spec?.byo?.deployment?.image : undefined;
  const isReady = accepted && deploymentReady;

  const ownerId = user_id || agent.metadata.annotations?.["kagent.dev/user-id"] || "";
  const isOwner = ownerId === currentUserId;
  const privateMode = typeof private_mode === "boolean"
    ? private_mode
    : agent.metadata.annotations?.["kagent.dev/private-mode"] !== "false";
  const protectedAgent = isEffectivelyProtectedCheck(protectedAgentNames, agent.metadata.name || "", isOwner);

  const category = agent.metadata.labels?.[LABEL_CATEGORY];
  const toolType = agent.metadata.labels?.[LABEL_TOOL_TYPE];
  const hasBadges = !!(toolType || category);

  const handleEditClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (isOwner) {
      router.push(`/agents/new?edit=true&name=${agent.metadata.name}&namespace=${agent.metadata.namespace}`);
      return;
    }

    router.push(`/agents/new?edit=true&readonly=true&name=${agent.metadata.name}&namespace=${agent.metadata.namespace}`);
  };

  const getStatusInfo = () => {
    if (!accepted) {
      return {
        message: "Agent not Accepted",
        className:"bg-red-500/10 text-red-600 dark:text-red-500"
      };
    }
    if (!deploymentReady) {
      return {
        message: "Agent not Ready",
        className:"bg-yellow-400/30 text-yellow-800 dark:bg-yellow-500/40 dark:text-yellow-200"
      };
    }
    return null;
  };

  const statusInfo = getStatusInfo();

  const cardContent = (
    <Card className={cn(
      "group relative transition-all duration-200 overflow-hidden min-h-[200px] border-l-2 border-l-transparent",
      isReady
        ? 'cursor-pointer hover:border-l-primary hover:shadow-md hover:-translate-y-0.5'
        : 'cursor-default'
    )}>
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2 relative z-30">
        <CardTitle className="flex items-center gap-2 flex-1 min-w-0">
          <KagentLogo className="h-5 w-5 flex-shrink-0" />
          <span className="truncate">{agent.metadata.name}</span>
          {protectedAgent && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Shield className="h-3 w-3 flex-shrink-0 text-muted-foreground" />
              </TooltipTrigger>
              <TooltipContent>
                This agent is protected and cannot be edited or deleted.
              </TooltipContent>
            </Tooltip>
          )}
        </CardTitle>
        <div className="relative z-30 opacity-0 group-hover:opacity-100 transition-opacity">
          <DropdownMenu>
            <DropdownMenuTrigger asChild onClick={(e) => { e.preventDefault(); e.stopPropagation(); }}>
              <Button
                variant="ghost"
                size="icon"
                aria-label="Agent options"
                className="bg-background/80 hover:bg-background shadow-sm"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
              <DropdownMenuItem
                onClick={handleEditClick}
                className="cursor-pointer"
              >
                {isOwner ? <Pencil className="mr-2 h-4 w-4" /> : <Eye className="mr-2 h-4 w-4" />}
                {isOwner ? "Edit" : "View"}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setMemoriesOpen(true);
                }}
                className="cursor-pointer"
              >
                <Brain className="mr-2 h-4 w-4" />
                View Memories
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={protectedAgent ? undefined : (e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setDeleteOpen(true);
                }}
                className={cn("cursor-pointer text-red-500 focus:text-red-500", protectedAgent && "opacity-50 cursor-not-allowed")}
                disabled={protectedAgent}
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col justify-between h-32 relative z-10">
        {hasBadges && (
          <div className="flex flex-wrap gap-1 mb-1">
            {toolType && <Badge variant="outline" className="text-[10px] capitalize">{toolType}</Badge>}
            {category && <Badge variant="outline" className="text-[10px] capitalize">{category}</Badge>}
          </div>
        )}
        <p className={cn("text-sm text-muted-foreground overflow-hidden", hasBadges ? "line-clamp-2" : "line-clamp-3")}>
          {agent.spec.description}
        </p>
        <div className="mt-auto flex flex-col gap-2">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <User className="h-3 w-3 flex-shrink-0" />
            <span className="truncate">{ownerId || "admin"}</span>
          </div>
          <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
            {isBYO ? (
              <span title={byoImage} className="truncate">Image: {byoImage}</span>
            ) : (
              <span className="truncate">{modelProvider} ({model})</span>
            )}
            {privateMode && (
              <span className="rounded-full bg-slate-200 px-2 py-0.5 text-[11px] font-medium text-slate-800 dark:bg-slate-700 dark:text-slate-100">
                Private
              </span>
            )}
          </div>
        </div>
      </CardContent>
      {statusInfo && (
        <div className={cn(
          "absolute bottom-0 left-0 right-0 z-20 py-1.5 px-4 text-right text-xs font-medium rounded-b-xl",
          statusInfo.className
        )}>
          {statusInfo.message}
        </div>
      )}

    </Card>
  );

  return (
    <>
      {isReady ? (
        <Link href={`/agents/${agent.metadata.namespace}/${agent.metadata.name}/chat`} passHref>
          {cardContent}
        </Link>
      ) : (
        cardContent
      )}

      <DeleteButton
        agentName={agent.metadata.name}
        namespace={agent.metadata.namespace || ''}
        externalOpen={deleteOpen}
        onExternalOpenChange={setDeleteOpen}
        disabled={protectedAgent}
      />

      <MemoriesDialog
        agentName={agent.metadata.name || ''}
        namespace={agent.metadata.namespace || ''}
        open={memoriesOpen}
        onOpenChange={setMemoriesOpen}
      />
    </>
  );
}
