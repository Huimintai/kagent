"use client";

import { useMemo, useState } from "react";
import { AgentGrid } from "@/components/AgentGrid";
import { AgentFilterToolbar } from "@/components/AgentFilterToolbar";
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
import { useUserStore } from "@/lib/userStore";
import { LABEL_CATEGORY, LABEL_TOOL_TYPE, LABEL_ROLE } from "@/lib/constants";
import type { PrivacyFilter } from "@/lib/constants";

const UNCATEGORIZED = "Uncategorized";

function getAgentCategory(agent: AgentResponse): string {
  return agent.agent.metadata.labels?.[LABEL_CATEGORY] || UNCATEGORIZED;
}

function toggleSetItem(prev: Set<string>, value: string): Set<string> {
  const next = new Set(prev);
  if (next.has(value)) next.delete(value);
  else next.add(value);
  return next;
}

export default function AgentList() {
  const { agents, loading, error } = useAgents();
  const currentUserId = useUserStore((state) => state.userId);

  const [searchTerm, setSearchTerm] = useState("");
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set());
  const [selectedToolTypes, setSelectedToolTypes] = useState<Set<string>>(new Set());
  const [selectedRoles, setSelectedRoles] = useState<Set<string>>(new Set());
  const [privacyFilter, setPrivacyFilter] = useState<PrivacyFilter>("all");
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({});

  // Extract unique values per dimension
  const allCategories = useMemo(() => {
    const s = new Set<string>();
    agents?.forEach((a) => s.add(getAgentCategory(a)));
    return s;
  }, [agents]);

  const allToolTypes = useMemo(() => {
    const s = new Set<string>();
    agents?.forEach((a) => {
      const v = a.agent.metadata.labels?.[LABEL_TOOL_TYPE];
      if (v) s.add(v);
    });
    return s;
  }, [agents]);

  const allRoles = useMemo(() => {
    const s = new Set<string>();
    agents?.forEach((a) => {
      const v = a.agent.metadata.labels?.[LABEL_ROLE];
      if (v) s.add(v);
    });
    return s;
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

  // Multi-dimensional filtering
  const filteredAgents = useMemo(() => {
    if (!agents) return [];
    return agents.filter((a) => {
      // Text search
      const search = searchTerm.toLowerCase();
      const ref = k8sRefUtils.toRef(a.agent.metadata.namespace || "", a.agent.metadata.name || "");
      const matchesSearch =
        !search ||
        ref.toLowerCase().includes(search) ||
        a.agent.metadata.name.toLowerCase().includes(search) ||
        a.agent.spec.description?.toLowerCase().includes(search);

      // Category
      const category = getAgentCategory(a);
      const matchesCategory = selectedCategories.size === 0 || selectedCategories.has(category);

      // Tool type
      const toolType = a.agent.metadata.labels?.[LABEL_TOOL_TYPE] || "";
      const matchesToolType = selectedToolTypes.size === 0 || (toolType !== "" && selectedToolTypes.has(toolType));

      // Role
      const role = a.agent.metadata.labels?.[LABEL_ROLE] || "";
      const matchesRole = selectedRoles.size === 0 || (role !== "" && selectedRoles.has(role));

      // Privacy
      const ownerId = a.user_id || a.agent.metadata.annotations?.["kagent.dev/user-id"] || "";
      const isOwner = ownerId === currentUserId;
      const isPrivate =
        typeof a.private_mode === "boolean"
          ? a.private_mode
          : a.agent.metadata.annotations?.["kagent.dev/private-mode"] !== "false";

      let matchesPrivacy = true;
      if (privacyFilter === "my") {
        matchesPrivacy = isOwner;
      }

      return matchesSearch && matchesCategory && matchesToolType && matchesRole && matchesPrivacy;
    });
  }, [agents, searchTerm, selectedCategories, selectedToolTypes, selectedRoles, privacyFilter, currentUserId]);

  // Group filtered agents by category
  const groupedAgents = useMemo(() => {
    const groups: Record<string, AgentResponse[]> = {};
    filteredAgents.forEach((a) => {
      const cat = getAgentCategory(a);
      if (!groups[cat]) groups[cat] = [];
      groups[cat].push(a);
    });
    return Object.entries(groups).sort(([a], [b]) => {
      if (a === UNCATEGORIZED) return 1;
      if (b === UNCATEGORIZED) return -1;
      return a.localeCompare(b);
    });
  }, [filteredAgents]);

  const toggleCategoryExpand = (cat: string) => {
    setExpandedCategories((prev) => ({ ...prev, [cat]: !prev[cat] }));
  };

  const hasActiveFilters =
    searchTerm !== "" ||
    selectedCategories.size > 0 ||
    selectedToolTypes.size > 0 ||
    selectedRoles.size > 0 ||
    privacyFilter !== "all";

  const handleClearAllFilters = () => {
    setSearchTerm("");
    setSelectedCategories(new Set());
    setSelectedToolTypes(new Set());
    setSelectedRoles(new Set());
    setPrivacyFilter("all");
  };

  if (error) return <ErrorState message={error} />;
  if (loading) return <LoadingState />;

  const hasMultipleCategories = allCategories.size > 1;

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
          <AgentFilterToolbar
            searchTerm={searchTerm}
            onSearchChange={setSearchTerm}
            privacyFilter={privacyFilter}
            onPrivacyFilterChange={setPrivacyFilter}
            filterSections={[
              {
                label: "Tool Type",
                values: allToolTypes,
                selected: selectedToolTypes,
                onToggle: (v) => setSelectedToolTypes((p) => toggleSetItem(p, v)),
              },
              {
                label: "Role",
                values: allRoles,
                selected: selectedRoles,
                onToggle: (v) => setSelectedRoles((p) => toggleSetItem(p, v)),
              },
              {
                label: "Category",
                values: allCategories,
                selected: selectedCategories,
                onToggle: (v) => setSelectedCategories((p) => toggleSetItem(p, v)),
              },
            ]}
            onClearAllFilters={handleClearAllFilters}
            hasActiveFilters={hasActiveFilters}
          />

          {/* Grouped display */}
          {filteredAgents.length === 0 ? (
            <div className="text-center py-12">
              <h3 className="text-lg font-medium mb-2">No agents found</h3>
              <p className="text-muted-foreground mb-4">Try adjusting your search or filters.</p>
              {hasActiveFilters && (
                <Button variant="outline" onClick={handleClearAllFilters}>
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
                    onClick={() => toggleCategoryExpand(category)}
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
            <AgentGrid agentResponse={filteredAgents} />
          )}
        </>
      )}
    </div>
  );
}
