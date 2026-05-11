import { AgentStat } from "@/types";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { getRelativeTimeString } from "@/lib/utils";

interface AgentLeaderboardProps {
  agents: AgentStat[];
}

function getMedal(rank: number): string {
  switch (rank) {
    case 1:
      return "\u{1F947}";
    case 2:
      return "\u{1F948}";
    case 3:
      return "\u{1F949}";
    default:
      return "";
  }
}

function extractName(agentId: string): string {
  const parts = agentId.split("/");
  const name = parts.length > 1 ? parts[parts.length - 1] : agentId;
  // Strip internal prefix used by the database layer
  return name.replace(/^dbci_agent__NS__/, "");
}

export function AgentLeaderboard({ agents }: AgentLeaderboardProps) {
  const maxScore = agents.length > 0 ? agents[0].score : 1;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Agent Leaderboard</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {agents.length === 0 && (
          <p className="text-sm text-muted-foreground">No agent activity yet.</p>
        )}
        {agents.map((agent, index) => {
          const rank = index + 1;
          const medal = getMedal(rank);
          const barWidth = maxScore > 0 ? (agent.score / maxScore) * 100 : 0;

          return (
            <div
              key={agent.agentId}
              className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50"
            >
              <div className="flex h-8 w-8 shrink-0 items-center justify-center text-sm font-semibold text-muted-foreground">
                {medal || `#${rank}`}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className="truncate text-sm font-medium text-foreground">
                    {extractName(agent.agentId)}
                  </p>
                  {agent.lastActiveAt && (
                    <span className="shrink-0 text-xs text-muted-foreground">
                      {getRelativeTimeString(agent.lastActiveAt)}
                    </span>
                  )}
                </div>
                <div className="mt-1.5 flex items-center gap-3">
                  <div className="relative h-2 flex-1 overflow-hidden rounded-full bg-muted">
                    <div
                      className="absolute inset-y-0 left-0 rounded-full bg-primary transition-all"
                      style={{ width: `${barWidth}%` }}
                    />
                  </div>
                  <span className="shrink-0 text-xs font-bold text-foreground">
                    {agent.score.toFixed(1)} pts
                  </span>
                </div>
                <div className="mt-0.5 flex items-center gap-3 text-xs text-muted-foreground">
                  <span>{agent.userCount} {agent.userCount === 1 ? "user" : "users"}</span>
                  <span>&middot;</span>
                  <span>{agent.sessionCount} sessions</span>
                  <span>&middot;</span>
                  <span>{agent.messageCount} msgs</span>
                </div>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
