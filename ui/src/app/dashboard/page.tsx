"use client";

import { useEffect, useState, useCallback } from "react";
import { AppPageFrame } from "@/components/layout/AppPageFrame";
import { PageHeader } from "@/components/layout/PageHeader";
import { StatCard } from "@/components/dashboard/StatCard";
import { AgentLeaderboard } from "@/components/dashboard/AgentLeaderboard";
import { HotMCPs } from "@/components/dashboard/HotMCPs";
import { StatsResponse, BaseResponse } from "@/types";
import { Bot, MessageSquare, Server, CalendarDays } from "lucide-react";

export default function DashboardPage() {
  const [stats, setStats] = useState<StatsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/stats?limit=10", { cache: "no-store" });
      if (!res.ok) {
        setError("Failed to load dashboard stats");
        return;
      }
      const json: BaseResponse<StatsResponse> = await res.json();
      if (json.error) {
        setError(json.error);
        return;
      }
      if (json.data) {
        setStats(json.data);
      }
    } catch {
      setError("Failed to load dashboard stats");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadStats();
  }, [loadStats]);

  return (
    <AppPageFrame ariaLabelledBy="dashboard-page-title" mainClassName="mx-auto max-w-6xl px-4 py-8 sm:px-6 sm:py-10">
      <PageHeader
        titleId="dashboard-page-title"
        title="Dashboard"
        description="Platform overview with top agents and tool servers."
        className="mb-6"
      />

      {loading && (
        <div className="flex items-center justify-center py-20">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
        </div>
      )}

      {error && !loading && (
        <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      )}

      {stats && !loading && (
        <div className="space-y-8">
          {/* Summary Cards */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              title="Total Agents"
              value={stats.summary.totalAgents}
              icon={<Bot className="h-5 w-5 text-muted-foreground" />}
            />
            <StatCard
              title="Total Sessions"
              value={stats.summary.totalSessions}
              icon={<MessageSquare className="h-5 w-5 text-muted-foreground" />}
            />
            <StatCard
              title="Sessions Today"
              value={stats.summary.sessionsToday}
              icon={<CalendarDays className="h-5 w-5 text-muted-foreground" />}
            />
            <StatCard
              title="Total MCPs"
              value={stats.summary.totalToolServers}
              icon={<Server className="h-5 w-5 text-muted-foreground" />}
            />
          </div>

          {/* Leaderboards */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <AgentLeaderboard agents={stats.topAgents} />
            <HotMCPs mcps={stats.topMCPs} />
          </div>
        </div>
      )}
    </AppPageFrame>
  );
}
