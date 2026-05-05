"use client";

import { useState, useMemo } from "react";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { useAgents } from "@/components/AgentsProvider";
import { LABEL_CATEGORY } from "@/lib/constants";

interface CategoryComboboxProps {
  value: string;
  onValueChange: (value: string) => void;
  disabled?: boolean;
}

export function CategoryCombobox({
  value,
  onValueChange,
  disabled = false,
}: CategoryComboboxProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const { agents } = useAgents();

  // Collect unique categories from existing agents
  const existingCategories = useMemo(() => {
    const cats = new Set<string>();
    agents?.forEach((a) => {
      const cat = a.agent.metadata.labels?.[LABEL_CATEGORY];
      if (cat) cats.add(cat);
    });
    return Array.from(cats).sort();
  }, [agents]);

  // Show the typed value as a create option if it doesn't match any existing category
  const trimmedSearch = search.trim().toLowerCase();
  const showCreateOption =
    trimmedSearch &&
    !existingCategories.some((c) => c.toLowerCase() === trimmedSearch);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
          disabled={disabled}
        >
          {value ? (
            <span className="capitalize">{value}</span>
          ) : (
            <span className="text-muted-foreground">Select or type a category...</span>
          )}
          <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-full p-0" align="start">
        <Command>
          <CommandInput
            placeholder="Search or create category..."
            value={search}
            onValueChange={setSearch}
          />
          <CommandList>
            <CommandEmpty>
              {trimmedSearch ? "Press enter or click below to create." : "No categories found."}
            </CommandEmpty>
            <CommandGroup>
              {/* Option to clear */}
              {value && (
                <CommandItem
                  value="__clear__"
                  onSelect={() => {
                    onValueChange("");
                    setSearch("");
                    setOpen(false);
                  }}
                >
                  <span className="text-muted-foreground italic">None (clear category)</span>
                </CommandItem>
              )}
              {existingCategories.map((cat) => (
                <CommandItem
                  key={cat}
                  value={cat}
                  onSelect={(currentValue) => {
                    onValueChange(currentValue === value ? "" : currentValue);
                    setSearch("");
                    setOpen(false);
                  }}
                >
                  <Check
                    className={cn(
                      "mr-2 h-4 w-4",
                      value === cat ? "opacity-100" : "opacity-0"
                    )}
                  />
                  <span className="capitalize">{cat}</span>
                </CommandItem>
              ))}
              {showCreateOption && (
                <CommandItem
                  value={trimmedSearch}
                  onSelect={() => {
                    onValueChange(trimmedSearch);
                    setSearch("");
                    setOpen(false);
                  }}
                >
                  <span>
                    Create &quot;<span className="font-medium">{trimmedSearch}</span>&quot;
                  </span>
                </CommandItem>
              )}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
