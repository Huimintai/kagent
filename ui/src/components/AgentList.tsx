"use client";

import { useMemo, useState } from "react";
import { AgentGrid } from "@/components/AgentGrid";
import { AgentCategoryFilter } from "@/components/AgentCategoryFilter";
import { Plus, ChevronDown, ChevronRight } from "lucide-react";
import KagentLogo from "@/components/kagent-logo";
import Link from "next/link";
import { ErrorState } from "./ErrorState";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { LoadingState } from "./LoadingState";
import { useAgents } from "./AgentsProvider";
import type { AgentResponse } from "@/types";
import { k8sRefUtils } from "@/lib/k8sUtils";

const CATEGORY_LABEL = "kagent.dev/category";
const UNCATEGORIZED = "Uncategorized";

function getAgentCategory(agent: AgentResponse): string {
  return agent.agent.metadata.labels?.[CATEGORY_LABEL] || UNCATEGORIZED;
}

export default function AgentList() {
  const { agents, loading, error } = useAgents();

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set());
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({});

  // Extract all unique categories
  const allCategories = useMemo(() => {
    const cats = new Set<string>();
    agents?.forEach((a) => cats.add(getAgentCategory(a)));
    return cats;
  }, [agents]);

  // Initialize expanded state for new categories
  useMemo(() => {
    setExpandedCategories((prev) => {
      const next = { ...prev };
      allCategories.forEach((cat) => {
        if (!(cat in next)) next[cat] = true;
      });
      return next;
    });
  }, [allCategories]);

  // Filter agents by search + selected categories
  const filteredAgents = useMemo(() => {
    if (!agents) return [];
    return agents.filter((a) => {
      const search = searchTerm.toLowerCase();
      const ref = k8sRefUtils.toRef(a.agent.metadata.namespace || "", a.agent.metadata.name || "");
      const matchesSearch =
        !search ||
        ref.toLowerCase().includes(search) ||
        a.agent.metadata.name.toLowerCase().includes(search) ||
        a.agent.spec.description?.toLowerCase().includes(search);

      const category = getAgentCategory(a);
      const matchesCategory = selectedCategories.size === 0 || selectedCategories.has(category);

      return matchesSearch && matchesCategory;
    });
  }, [agents, searchTerm, selectedCategories]);

  // Group filtered agents by category
  const groupedAgents = useMemo(() => {
    const groups: Record<string, AgentResponse[]> = {};
    filteredAgents.forEach((a) => {
      const cat = getAgentCategory(a);
      if (!groups[cat]) groups[cat] = [];
      groups[cat].push(a);
    });
    // Sort categories alphabetically, but put "Uncategorized" last
    return Object.entries(groups).sort(([a], [b]) => {
      if (a === UNCATEGORIZED) return 1;
      if (b === UNCATEGORIZED) return -1;
      return a.localeCompare(b);
    });
  }, [filteredAgents]);

  const toggleCategory = (cat: string) => {
    setExpandedCategories((prev) => ({ ...prev, [cat]: !prev[cat] }));
  };

  const handleToggleFilter = (category: string) => {
    setSelectedCategories((prev) => {
      const next = new Set(prev);
      if (next.has(category)) next.delete(category);
      else next.add(category);
      return next;
    });
  };

  if (error) return <ErrorState message={error} />;
  if (loading) return <LoadingState />;

  const hasMultipleCategories = allCategories.size > 1;
  const hasFilters = searchTerm || selectedCategories.size > 0;

  return (
    <div className="mt-12 mx-auto max-w-6xl px-6">
      <div className="flex justify-between items-center mb-8">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold">Agents</h1>
          {agents && agents.length > 0 && (
            <Badge variant="secondary" className="font-mono text-xs">
              {filteredAgents.length}/{agents.length}
            </Badge>
          )}
        </div>
      </div>

      {agents?.length === 0 ? (
        <div className="text-center py-12">
          <KagentLogo className="h-16 w-16 mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No agents yet</h3>
          <p className="mb-6">Create your first agent to get started</p>
          <Button className="bg-violet-500 hover:bg-violet-600" asChild>
            <Link href="/agents/new">
              <Plus className="h-4 w-4 mr-2" />
              Create New Agent
            </Link>
          </Button>
        </div>
      ) : (
        <>
          {/* Search + Category Filter */}
          {(hasMultipleCategories || (agents && agents.length > 5)) && (
            <AgentCategoryFilter
              categories={allCategories}
              selectedCategories={selectedCategories}
              searchTerm={searchTerm}
              onToggleCategory={handleToggleFilter}
              onSelectAll={() => setSelectedCategories(new Set(allCategories))}
              onClearAll={() => setSelectedCategories(new Set())}
              onSearchChange={setSearchTerm}
            />
          )}

          {/* Grouped display */}
          {filteredAgents.length === 0 ? (
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">No agents found</h3>
              <p className="text-muted-foreground mb-4">Try adjusting your search or filters.</p>
              {hasFilters && (
                <Button
                  variant="outline"
                  onClick={() => {
                    setSearchTerm("");
                    setSelectedCategories(new Set());
                  }}
                >
                  Clear Filters
                </Button>
              )}
            </div>
          ) : hasMultipleCategories ? (
            <div className="space-y-4">
              {groupedAgents.map(([category, categoryAgents]) => (
                <div key={category} className="border rounded-lg overflow-hidden bg-card shadow-sm">
                  <div
                    className="flex items-center justify-between p-4 bg-secondary/50 cursor-pointer hover:bg-secondary/70 transition-colors"
                    onClick={() => toggleCategory(category)}
                  >
                    <div className="flex items-center gap-2">
                      {expandedCategories[category] ? (
                        <ChevronDown className="w-4 h-4" />
                      ) : (
                        <ChevronRight className="w-4 h-4" />
                      )}
                      <h3 className="font-semibold capitalize text-sm">{category}</h3>
                      <Badge variant="secondary" className="font-mono text-xs">
                        {categoryAgents.length}
                      </Badge>
                    </div>
                  </div>
                  {expandedCategories[category] && (
                    <div className="p-4 border-t">
                      <AgentGrid agentResponse={categoryAgents} />
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            /* Single category or no categories — flat grid */
            <AgentGrid agentResponse={filteredAgents} />
          )}
        </>
      )}
    </div>
  );
}
