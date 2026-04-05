import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

interface AgentCategoryFilterProps {
  categories: Set<string>;
  selectedCategories: Set<string>;
  searchTerm: string;
  onToggleCategory: (category: string) => void;
  onSelectAll: () => void;
  onClearAll: () => void;
  onSearchChange: (value: string) => void;
}

export function AgentCategoryFilter({
  categories,
  selectedCategories,
  searchTerm,
  onToggleCategory,
  onSelectAll,
  onClearAll,
  onSearchChange,
}: AgentCategoryFilterProps) {
  return (
    <div className="space-y-4 mb-6">
      <div className="relative">
        <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder="Search agents by name or description..."
          value={searchTerm}
          onChange={(e) => onSearchChange(e.target.value)}
          className="pl-10"
        />
      </div>
      {categories.size > 1 && (
        <div className="p-4 border rounded-md bg-secondary/10">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-medium">Filter by Category</h3>
            <div className="flex gap-2">
              <Button variant="ghost" size="sm" onClick={onClearAll}>
                Clear All
              </Button>
              <Button variant="ghost" size="sm" onClick={onSelectAll}>
                Select All
              </Button>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {Array.from(categories)
              .sort()
              .map((category) => (
                <Badge
                  key={category}
                  variant={selectedCategories.has(category) ? "default" : "outline"}
                  className="cursor-pointer capitalize"
                  onClick={() => onToggleCategory(category)}
                >
                  {category}
                </Badge>
              ))}
          </div>
        </div>
      )}
    </div>
  );
}
