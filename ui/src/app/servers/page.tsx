"use client";

import { useState, useEffect } from "react";
import { Server, ChevronDown, ChevronRight, FunctionSquare } from "lucide-react";
import { ToolServerResponse } from "@/types";
import { getServers } from "../actions/servers";
import Link from "next/link";
import { toast } from "sonner";

export default function ServersPage() {
  const [servers, setServers] = useState<ToolServerResponse[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [expandedServers, setExpandedServers] = useState<Set<string>>(new Set());

  useEffect(() => {
    fetchServers();
  }, []);

  const fetchServers = async () => {
    try {
      setIsLoading(true);
      const serversResponse = await getServers();
      if (!serversResponse.error && serversResponse.data) {
        const sortedServers = [...serversResponse.data].sort((a, b) => {
          return (a.ref || '').localeCompare(b.ref || '');
        });
        setServers(sortedServers);
        setExpandedServers(new Set());
      } else {
        console.error("Failed to fetch servers:", serversResponse);
        toast.error(serversResponse.error || "Failed to fetch servers data.");
      }
    } catch (error) {
      console.error("Error fetching servers:", error);
      toast.error("An error occurred while fetching servers.");
    } finally {
      setIsLoading(false);
    }
  };

  const toggleServer = (serverName: string) => {
    setExpandedServers(prev => {
      const newSet = new Set(prev);
      if (newSet.has(serverName)) {
        newSet.delete(serverName);
      } else {
        newSet.add(serverName);
      }
      return newSet;
    });
  };

  return (
    <div className="mt-12 mx-auto max-w-6xl px-6">
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold">MCP Servers</h1>
          <Link href="/tools" className="text-blue-600 hover:text-blue-800 text-sm">
            View Tools →
          </Link>
        </div>
        {servers.length > 0 && (
          <span className="text-sm text-muted-foreground">{servers.length} servers</span>
        )}
      </div>

      {isLoading ? (
        <div className="flex flex-col items-center justify-center h-[200px] border rounded-lg bg-secondary/5">
          <div className="animate-pulse h-6 w-6 rounded-full bg-primary/10 mb-4"></div>
          <p className="text-muted-foreground">Loading servers...</p>
        </div>
      ) : servers.length > 0 ? (
        <div className="space-y-4">
          {servers.map((server) => {
            if (!server.ref) return null;
            const serverName: string = server.ref;
            const isExpanded = expandedServers.has(serverName);

            return (
              <div key={server.ref} className="border rounded-md overflow-hidden">
                <div
                  className="bg-secondary/10 p-4 cursor-pointer"
                  onClick={() => toggleServer(serverName)}
                >
                  <div className="flex items-center gap-3">
                    {isExpanded ? <ChevronDown className="h-5 w-5" /> : <ChevronRight className="h-5 w-5" />}
                    <div>
                      <div className="font-medium">{server.ref}</div>
                    </div>
                  </div>
                </div>

                {isExpanded && (
                  <div className="p-4">
                    {server.discoveredTools && server.discoveredTools.length > 0 ? (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                        {server.discoveredTools
                          .sort((a, b) => (a.name || "").localeCompare(b.name || ""))
                          .map((tool) => (
                            <div key={tool.name} className="p-3 border rounded-md hover:bg-secondary/5 transition-colors">
                              <div className="flex items-start gap-2">
                                <FunctionSquare className="h-4 w-4 text-blue-500 mt-0.5" />
                                <div>
                                  <div className="font-medium text-sm">{tool.name}</div>
                                  <div className="text-xs text-muted-foreground mt-1">{tool.description}</div>
                                </div>
                              </div>
                            </div>
                          ))}
                      </div>
                    ) : (
                      <div className="text-center p-4 text-sm text-muted-foreground">No tools available for this MCP server.</div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center h-[300px] text-center p-4 border rounded-lg bg-secondary/5">
          <Server className="h-12 w-12 text-muted-foreground mb-4 opacity-20" />
          <h3 className="font-medium text-lg">No MCP servers connected</h3>
          <p className="text-muted-foreground mt-1 mb-4">MCP servers are managed by the platform administrator.</p>
        </div>
      )}
    </div>
  );
}
