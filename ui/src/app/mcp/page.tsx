"use client";

import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { McpServersView } from "@/components/mcp/McpServersView";
import type { ToolServerResponse, BaseResponse } from "@/types";

const POLL_INTERVAL_MS = 2000;
const POLL_MAX_MS = 30000;

async function fetchServersFresh(): Promise<ToolServerResponse[]> {
  const res = await fetch("/api/toolservers", { cache: "no-store" });
  if (!res.ok) return [];
  const data: BaseResponse<ToolServerResponse[]> = await res.json();
  return data.data ?? [];
}

function McpPageContent() {
  const searchParams = useSearchParams();
  const newServerName = searchParams.get("new") ?? undefined;

  const [servers, setServers] = useState<ToolServerResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingServers, setPendingServers] = useState<Set<string>>(new Set());
  const pollTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollStartRef = useRef<number>(0);

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearTimeout(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await fetchServersFresh();
      const sorted = [...data].sort((a, b) => (a.ref || "").localeCompare(b.ref || ""));
      setServers(sorted);
      if (newServerName) {
        const match = sorted.find((s) => s.ref === newServerName || s.ref?.endsWith(`/${newServerName}`));
        if (match?.ref && (match.discoveredTools?.length ?? 0) === 0) {
          setPendingServers(new Set([match.ref]));
          pollStartRef.current = Date.now();
        }
      }
    } catch {
      setLoadError("Failed to load MCP servers");
      setServers([]);
      setPendingServers(new Set());
    }
    setLoading(false);
  }, [newServerName]);

  // Poll directly (bypassing server action cache) until tools arrive or timeout
  useEffect(() => {
    if (pendingServers.size === 0) {
      stopPolling();
      return;
    }
    if (Date.now() - pollStartRef.current >= POLL_MAX_MS) {
      setPendingServers(new Set());
      return;
    }
    const pendingRef = [...pendingServers][0];
    const timerId = setTimeout(async () => {
      const data = await fetchServersFresh();
      const sorted = data.sort((a, b) => (a.ref || "").localeCompare(b.ref || ""));
      const target = sorted.find((s) => s.ref === pendingRef);
      if (!target || (target.discoveredTools?.length ?? 0) > 0) {
        setServers(sorted);
        setPendingServers(new Set());
      } else {
        setServers(sorted);
        // New Set instance triggers the effect again for the next poll tick
        setPendingServers(new Set([pendingRef]));
      }
    }, POLL_INTERVAL_MS);
    pollTimerRef.current = timerId;

    return () => clearTimeout(timerId);
  }, [pendingServers, stopPolling]);

  useEffect(() => {
    const raf = requestAnimationFrame(() => { void load(); });
    return () => cancelAnimationFrame(raf);
  }, [load]);

  return (
    <AppPageFrame ariaLabelledBy="mcp-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="mcp-page-title"
        title="MCP & tools"
        description="Add MCP servers to your cluster, then search or expand each server to see the tools agents can use."
        className="mb-6"
      />

      <McpServersView servers={servers} isLoading={loading} loadError={loadError} onRefresh={load} pendingServers={pendingServers} />
    </AppPageFrame>
  );
}

export default function McpPage() {
  return (
    <Suspense>
      <McpPageContent />
    </Suspense>
  );
}
