import type { Session } from "@/types";
import ChatItem from "@/components/sidebars/ChatItem";
import { SidebarGroup, SidebarMenu, SidebarMenuSub } from "../ui/sidebar";
import { Collapsible } from "@radix-ui/react-collapsible";
import { ChevronRight } from "lucide-react";
import { CollapsibleContent, CollapsibleTrigger } from "../ui/collapsible";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "../ui/tooltip";
import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";

interface ChatGroupProps {
  title: string;
  sessions: Session[];
  onDeleteSession: (sessionId: string) => Promise<void>;
  onDownloadSession: (sessionId: string) => Promise<void>;
  agentName: string;
  agentNamespace: string;
  hideSessionDelete?: boolean;
  canPin?: boolean;
  pinnedSessionIds?: Set<string>;
  onPin?: (sessionId: string, pinned: boolean) => Promise<void>;
  defaultOpen?: boolean;
}

// The sessions are grouped by today, yesterday, and older
const ChatGroup = ({ title, sessions, onDeleteSession, onDownloadSession, agentName, agentNamespace, hideSessionDelete, canPin, pinnedSessionIds, onPin, defaultOpen }: ChatGroupProps) => {
  const pathname = usePathname();
  const hasActiveSession = sessions.some(s =>
    pathname === `/agents/${agentNamespace}/${agentName}/chat/${s.id}`
  );
  const isOpenByDefault = defaultOpen !== undefined ? defaultOpen : title.toLocaleLowerCase() === "today";
  const [open, setOpen] = useState(isOpenByDefault || hasActiveSession);

  // Force open whenever the active session belongs to this group
  useEffect(() => {
    if (hasActiveSession) setOpen(true);
  }, [hasActiveSession]);

  return (
    <SidebarGroup>
      <SidebarMenu>
        <Collapsible key={title} open={open} onOpenChange={setOpen} className="group/collapsible w-full">
          <div className="w-full">
            {title === "Pinned" ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <CollapsibleTrigger className="flex items-center justify-between w-full rounded-md p-2 pr-[9px] text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground">
                      <span>{title}</span>
                      <ChevronRight className="h-4 w-4 shrink-0 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                    </CollapsibleTrigger>
                  </TooltipTrigger>
                  <TooltipContent side="right">
                    Pinned sample chats shared by agent owner
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              <CollapsibleTrigger className="flex items-center justify-between w-full rounded-md p-2 pr-[9px] text-sm hover:bg-sidebar-accent hover:text-sidebar-accent-foreground">
                <span>{title}</span>
                <ChevronRight className="h-4 w-4 shrink-0 transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
              </CollapsibleTrigger>
            )}
          </div>
          <CollapsibleContent>
            <SidebarMenuSub className="mx-0 px-0 ml-2 pl-2">
              {sessions.map((session) => (
                <ChatItem
                  key={session.id}
                  sessionId={session.id!}
                  agentName={agentName}
                  agentNamespace={agentNamespace}
                  onDelete={onDeleteSession}
                  sessionName={session.name}
                  onDownload={onDownloadSession}
                  createdAt={session.created_at}
                  hideDelete={hideSessionDelete}
                  pinned={pinnedSessionIds ? pinnedSessionIds.has(session.id!) : session.pinned}
                  canPin={canPin}
                  onPin={onPin}
                />
              ))}
            </SidebarMenuSub>
          </CollapsibleContent>
        </Collapsible>
      </SidebarMenu>
    </SidebarGroup>
  );
};

export default ChatGroup;
