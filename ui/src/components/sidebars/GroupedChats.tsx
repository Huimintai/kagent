"use client";
import { useMemo, useState } from "react";
import ChatGroup from "./SessionGroup";
import type { AgentResponse, Session } from "@/types";
import { isToday, isYesterday } from "date-fns";
import { EmptyState } from "./EmptyState";
import { deleteSession, getSessionTasks, patchSession } from "@/app/actions/sessions";
import { Button } from "@/components/ui/button";
import { PlusCircle } from "lucide-react";
import { toast } from "sonner";

interface GroupedChatsProps {
  agentName: string;
  agentNamespace: string;
  sessions: Session[];
  /** Sandbox agents use a single persistent chat; hide "New Chat". */
  hideNewChat?: boolean;
  /** Sandbox agents cannot delete their only session from the UI. */
  hideSessionDelete?: boolean;
  currentAgent: AgentResponse;
  currentUserId: string;
}

export default function GroupedChats({ agentName, agentNamespace, sessions, hideNewChat, hideSessionDelete, currentAgent, currentUserId }: GroupedChatsProps) {
  // Optimistic state: track per-session overrides and deleted IDs separately
  // so we never need to sync props → state inside an effect.
  const [overrides, setOverrides] = useState<Record<string, Partial<Session>>>({});
  const [deletedIds, setDeletedIds] = useState<Set<string>>(new Set());

  // Compute the effective session list by merging props with optimistic overrides.
  const localSessions = useMemo(() => {
    return sessions
      .filter(s => !deletedIds.has(s.id))
      .map(s => overrides[s.id] ? { ...s, ...overrides[s.id] } : s);
  }, [sessions, overrides, deletedIds]);

  const groupedChats = useMemo(() => {
    // First split into pinned vs unpinned
    const pinned: Session[] = [];
    const unpinnedAll: Session[] = [];

    localSessions.forEach(session => {
      if (session.pinned === true) {
        pinned.push(session);
      } else {
        unpinnedAll.push(session);
      }
    });

    // Only show unpinned sessions that belong to the current user
    const unpinned = unpinnedAll.filter(session => session.user_id === currentUserId);

    const groups: {
      today: Session[];
      yesterday: Session[];
      older: Session[];
    } = {
      today: [],
      yesterday: [],
      older: [],
    };

    // Apply date grouping only to unpinned sessions
    unpinned.forEach(session => {
      const date = new Date(session.created_at);
      if (isToday(date)) {
        groups.today.push(session);
      } else if (isYesterday(date)) {
        groups.yesterday.push(session);
      } else {
        groups.older.push(session);
      }
    });

    const sortChats = (sessions: Session[]) =>
      sessions.sort((a, b) => {
        const getLatestTimestamp = (session: Session) => {
          return new Date(session.created_at).getTime();
        };

        return getLatestTimestamp(b) - getLatestTimestamp(a);
      });

    return {
      pinned: sortChats(pinned),
      today: sortChats(groups.today),
      yesterday: sortChats(groups.yesterday),
      older: sortChats(groups.older),
    };
  }, [localSessions, currentUserId]);

  const onDeleteClick = async (sessionId: string) => {
    try {
      // Immediately hide from local state
      setDeletedIds(prev => new Set([...prev, sessionId]));

      // Then delete from server
      await deleteSession(sessionId);
    } catch (error) {
      console.error("Error deleting session:", error);
      // If there's an error, restore the session in the UI
      setDeletedIds(prev => {
        const next = new Set(prev);
        next.delete(sessionId);
        return next;
      });
    }
  };

  const onDownloadClick = async (sessionId: string) => {
    toast.promise(
      getSessionTasks(String(sessionId)).then(messages => {
        const blob = new Blob([JSON.stringify(messages, null, 2)], { type: "application/json" });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `session-${sessionId}.json`;
        a.click();
        URL.revokeObjectURL(url);
        return messages;
      }),
      {
        loading: "Downloading session...",
        success: "Session downloaded successfully",
        error: "Failed to download session",
      }
    );
  };

  const onPin = async (sessionId: string, pinned: boolean) => {
    // Optimistically update local state
    const previousOverride = overrides[sessionId];
    setOverrides(prev => ({ ...prev, [sessionId]: { ...prev[sessionId], pinned } }));

    try {
      await patchSession(sessionId, { pinned });
    } catch (error) {
      console.error("Error pinning session:", error);
      // Revert on error
      setOverrides(prev => {
        const next = { ...prev };
        if (previousOverride !== undefined) {
          next[sessionId] = previousOverride;
        } else {
          delete next[sessionId];
        }
        return next;
      });
      toast.error("Failed to update pin state");
    }
  };

  const handleNewChat = () => {
    // Force a full page reload instead of client-side navigation
    window.location.href = `/agents/${agentNamespace}/${agentName}/chat`;
  };

  const canPin = currentAgent.user_id === currentUserId && !currentAgent.private_mode;

  const hasNoSessions =
    !groupedChats.pinned.length &&
    !groupedChats.today.length &&
    !groupedChats.yesterday.length &&
    !groupedChats.older.length;

  return (
    <>
      {!hideNewChat && (
      <div className="mb-4 px-2">
        <Button
          variant="secondary"
          className="w-full flex items-center justify-center gap-2"
          onClick={handleNewChat}
        >
          <PlusCircle size={16} />
          New Chat
        </Button>
      </div>
      )}

      {hasNoSessions || localSessions.length === 0 ? (
        <EmptyState variant={hideNewChat ? "singleChat" : "default"} />
      ) : (
        <>
          {groupedChats.pinned.length > 0 && (
            <ChatGroup
              title="Pinned"
              sessions={groupedChats.pinned}
              agentName={agentName}
              agentNamespace={agentNamespace}
              onDeleteSession={(sessionId) => onDeleteClick(sessionId)}
              onDownloadSession={(sessionId) => onDownloadClick(sessionId)}
              hideSessionDelete={true}
              canPin={canPin}
              onPin={onPin}
              defaultOpen={true}
            />
          )}
          {groupedChats.today.length > 0 && (
            <ChatGroup
              title="Today"
              sessions={groupedChats.today}
              agentName={agentName}
              agentNamespace={agentNamespace}
              onDeleteSession={(sessionId) => onDeleteClick(sessionId)}
              onDownloadSession={(sessionId) => onDownloadClick(sessionId)}
              hideSessionDelete={hideSessionDelete}
              canPin={canPin}
              onPin={onPin}
            />
          )}
          {groupedChats.yesterday.length > 0 && (
            <ChatGroup
              title="Yesterday"
              sessions={groupedChats.yesterday}
              agentName={agentName}
              agentNamespace={agentNamespace}
              onDeleteSession={(sessionId) => onDeleteClick(sessionId)}
              onDownloadSession={(sessionId) => onDownloadClick(sessionId)}
              hideSessionDelete={hideSessionDelete}
              canPin={canPin}
              onPin={onPin}
            />
          )}
          {groupedChats.older.length > 0 && (
            <ChatGroup
              title="Older"
              sessions={groupedChats.older}
              agentName={agentName}
              agentNamespace={agentNamespace}
              onDeleteSession={(sessionId) => onDeleteClick(sessionId)}
              onDownloadSession={(sessionId) => onDownloadClick(sessionId)}
              hideSessionDelete={hideSessionDelete}
              canPin={canPin}
              onPin={onPin}
            />
          )}
        </>
      )}
    </>
  );
}
