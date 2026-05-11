"use client";

import { useEffect, useState, useRef } from "react";
import { Trash2, Send } from "lucide-react";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";
import { Identicon } from "@/components/Identicon";
import { Button } from "@/components/ui/button";
import { useUserStore } from "@/lib/userStore";
import { getAgentComments, createAgentComment, deleteAgentComment } from "@/app/actions/comments";
import type { AgentComment } from "@/types";

interface AgentCommentsProps {
  namespace: string;
  agentName: string;
  isOwner?: boolean;
}

function formatUserDisplay(userId: string): string {
  const atIndex = userId.indexOf("@");
  if (atIndex > 0) {
    return userId.substring(0, atIndex);
  }
  return userId;
}

export function AgentComments({ namespace, agentName, isOwner = false }: AgentCommentsProps) {
  const [comments, setComments] = useState<AgentComment[]>([]);
  const [content, setContent] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const currentUserId = useUserStore((state) => state.userId);

  const MAX_CHARS = 500;
  const isMounted = useRef(true);

  useEffect(() => {
    isMounted.current = true;
    let cancelled = false;

    async function load() {
      setIsLoading(true);
      try {
        const result = await getAgentComments(namespace, agentName);
        if (!cancelled && result.data) {
          const sorted = [...result.data].sort(
            (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
          );
          setComments(sorted);
        }
      } catch {
        // Silently fail on load — show empty state
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    load();

    return () => {
      cancelled = true;
      isMounted.current = false;
    };
  }, [namespace, agentName]);

  const handleSubmit = async () => {
    const trimmed = content.trim();
    if (!trimmed || isSubmitting) return;

    setIsSubmitting(true);

    // Optimistic update
    const optimisticComment: AgentComment = {
      id: `temp-${Date.now()}`,
      agentId: `${namespace}/${agentName}`,
      userId: currentUserId,
      content: trimmed,
      createdAt: new Date().toISOString(),
    };

    setComments((prev) => [optimisticComment, ...prev]);
    setContent("");

    try {
      const result = await createAgentComment(namespace, agentName, trimmed);
      if (result.error) {
        // Revert optimistic update
        setComments((prev) => prev.filter((c) => c.id !== optimisticComment.id));
        setContent(trimmed);
        toast.error(result.error);
      } else if (result.data) {
        // Replace optimistic comment with real one
        setComments((prev) =>
          prev.map((c) => (c.id === optimisticComment.id ? result.data! : c))
        );
      }
    } catch {
      // Revert optimistic update
      setComments((prev) => prev.filter((c) => c.id !== optimisticComment.id));
      setContent(trimmed);
      toast.error("Failed to post comment");
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async (commentId: string) => {
    const commentToDelete = comments.find((c) => c.id === commentId);
    if (!commentToDelete) return;

    // Optimistic removal
    setComments((prev) => prev.filter((c) => c.id !== commentId));

    try {
      const result = await deleteAgentComment(namespace, agentName, commentId);
      if (result.error) {
        // Revert
        setComments((prev) => {
          const restored = [...prev, commentToDelete].sort(
            (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
          );
          return restored;
        });
        toast.error(result.error);
      }
    } catch {
      // Revert
      setComments((prev) => {
        const restored = [...prev, commentToDelete].sort(
          (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
        );
        return restored;
      });
      toast.error("Failed to delete comment");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div className="flex flex-col gap-3 px-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-foreground">
          {isOwner ? "Comments" : "Leave Feedback"}
        </span>
        {isOwner && comments.length > 0 && (
          <span className="text-xs text-muted-foreground bg-muted rounded-full px-2 py-0.5">
            {comments.length}
          </span>
        )}
      </div>

      {/* Input */}
      <div className="flex flex-col gap-1.5">
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value.slice(0, MAX_CHARS))}
          onKeyDown={handleKeyDown}
          placeholder={isOwner ? "Leave a comment..." : "Leave feedback for the agent owner..."}
          rows={2}
          className="w-full resize-none rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          disabled={isSubmitting}
        />
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            {content.length}/{MAX_CHARS}
          </span>
          <Button
            size="sm"
            variant="default"
            onClick={handleSubmit}
            disabled={!content.trim() || isSubmitting}
            className="h-7 px-2.5 text-xs"
          >
            <Send className="h-3 w-3 mr-1" />
            Post
          </Button>
        </div>
      </div>

      {/* Comments list — only visible to agent owner */}
      {isOwner && (
        <div className="flex flex-col gap-2 max-h-[300px] overflow-y-auto">
        {isLoading && comments.length === 0 && (
          <p className="text-xs text-muted-foreground italic">Loading comments...</p>
        )}
        {!isLoading && comments.length === 0 && (
          <p className="text-xs text-muted-foreground italic">No comments yet. Be the first!</p>
        )}
        {comments.map((comment) => (
          <div
            key={comment.id}
            className="flex gap-2 rounded-md border border-border bg-muted/30 p-2"
          >
            <Identicon value={comment.userId} size={24} className="flex-shrink-0 mt-0.5" />
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between gap-1">
                <span className="text-xs font-medium text-foreground truncate">
                  {formatUserDisplay(comment.userId)}
                </span>
                <div className="flex items-center gap-1 flex-shrink-0">
                  <span className="text-[10px] text-muted-foreground">
                    {formatDistanceToNow(new Date(comment.createdAt), { addSuffix: true })}
                  </span>
                  {comment.userId === currentUserId && (
                    <button
                      onClick={() => handleDelete(comment.id)}
                      className="text-muted-foreground hover:text-destructive transition-colors p-0.5"
                      aria-label="Delete comment"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  )}
                </div>
              </div>
              <p className="text-xs text-muted-foreground mt-0.5 break-words whitespace-pre-wrap">
                {comment.content}
              </p>
            </div>
          </div>
        ))}
      </div>
      )}
      {!isOwner && (
        <p className="text-xs text-muted-foreground italic">
          Your feedback will be visible to the agent owner.
        </p>
      )}
    </div>
  );
}
