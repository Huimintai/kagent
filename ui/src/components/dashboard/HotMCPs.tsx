import { ToolServerStat } from "@/types";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";

interface HotMCPsProps {
  mcps: ToolServerStat[];
}

function extractName(fullName: string): string {
  const parts = fullName.split("/");
  return parts.length > 1 ? parts[parts.length - 1] : fullName;
}

function isRecentlyConnected(lastConnected?: string): boolean {
  if (!lastConnected) return false;
  const oneHourAgo = Date.now() - 60 * 60 * 1000;
  return new Date(lastConnected).getTime() > oneHourAgo;
}

export function HotMCPs({ mcps }: HotMCPsProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Hot MCPs</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {mcps.length === 0 && (
          <p className="text-sm text-muted-foreground">No tool servers found.</p>
        )}
        {mcps.map((mcp, index) => {
          const rank = index + 1;
          const fire = rank <= 3 ? "\u{1F525}" : "";
          const active = isRecentlyConnected(mcp.lastConnected);

          return (
            <div
              key={mcp.name}
              className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50"
            >
              <div className="flex h-8 w-8 shrink-0 items-center justify-center text-sm">
                {fire || `#${rank}`}
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <p className="truncate text-sm font-medium text-foreground">
                    {extractName(mcp.name)}
                  </p>
                  <span className="shrink-0 rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-foreground">
                    {mcp.agentCount} {mcp.agentCount === 1 ? "agent" : "agents"}
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-1.5 shrink-0">
                <span
                  className={`inline-block h-2 w-2 rounded-full ${
                    active ? "bg-green-500" : "bg-gray-400"
                  }`}
                />
                <span className="text-xs text-muted-foreground">
                  {active ? "active" : "idle"}
                </span>
              </div>
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}
