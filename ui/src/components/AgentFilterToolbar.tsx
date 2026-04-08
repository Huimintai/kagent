"use client";

import { Search, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { PrivacyFilter } from "@/lib/constants";

interface FilterSection {
  label: string;
  values: Set<string>;
  selected: Set<string>;
  onToggle: (value: string) => void;
}

interface AgentFilterToolbarProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  privacyFilter: PrivacyFilter;
  onPrivacyFilterChange: (value: PrivacyFilter) => void;
  filterSections: FilterSection[];
  onClearAllFilters: () => void;
  hasActiveFilters: boolean;
}

export function AgentFilterToolbar({
  searchTerm,
  onSearchChange,
  privacyFilter,
  onPrivacyFilterChange,
  filterSections,
  onClearAllFilters,
  hasActiveFilters,
}: AgentFilterToolbarProps) {
  const hasFilterBadges = filterSections.some((s) => s.values.size > 0);

  return (
    <div className="space-y-3 mb-6">
      {/* Row 1: Search + Privacy toggle */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Search agents by name or description..."
            value={searchTerm}
            onChange={(e) => onSearchChange(e.target.value)}
            className="pl-10"
          />
        </div>
        <Tabs value={privacyFilter} onValueChange={(v) => onPrivacyFilterChange(v as PrivacyFilter)}>
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="public">Public</TabsTrigger>
            <TabsTrigger value="my">My</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Row 2: Classification badge filters */}
      {hasFilterBadges && (
        <div className="flex flex-wrap items-start gap-4 p-3 border rounded-md bg-secondary/10">
          {filterSections.map(
            (section) =>
              section.values.size > 0 && (
                <div key={section.label} className="flex items-center gap-2 flex-wrap">
                  <span className="text-xs font-medium text-muted-foreground whitespace-nowrap">
                    {section.label}:
                  </span>
                  {Array.from(section.values)
                    .sort()
                    .map((value) => (
                      <Badge
                        key={value}
                        variant={section.selected.has(value) ? "default" : "outline"}
                        className="cursor-pointer capitalize text-xs"
                        onClick={() => section.onToggle(value)}
                      >
                        {value}
                      </Badge>
                    ))}
                </div>
              )
          )}
          {hasActiveFilters && (
            <Button variant="ghost" size="sm" onClick={onClearAllFilters} className="ml-auto h-7 text-xs">
              <X className="h-3 w-3 mr-1" />
              Clear filters
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
