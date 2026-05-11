"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useState } from "react";

interface AddServerDialogProps {
  open: boolean;
  supportedToolServerTypes: string[];
  onOpenChange: (open: boolean) => void;
  onAddServer: (data: { name: string; namespace: string; url: string; type: string }) => void;
}

export function AddServerDialog({ open, supportedToolServerTypes, onOpenChange, onAddServer }: AddServerDialogProps) {
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [url, setUrl] = useState("");
  const [type, setType] = useState(supportedToolServerTypes[0] || "");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onAddServer({ name, namespace, url, type });
    setName("");
    setUrl("");
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add MCP Server</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <Label htmlFor="server-name">Name</Label>
            <Input id="server-name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div>
            <Label htmlFor="server-namespace">Namespace</Label>
            <Input id="server-namespace" value={namespace} onChange={(e) => setNamespace(e.target.value)} required />
          </div>
          <div>
            <Label htmlFor="server-url">URL</Label>
            <Input id="server-url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://..." required />
          </div>
          {supportedToolServerTypes.length > 0 && (
            <div>
              <Label htmlFor="server-type">Type</Label>
              <Select value={type} onValueChange={setType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {supportedToolServerTypes.map((t) => (
                    <SelectItem key={t} value={t}>{t}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
            <Button type="submit">Add Server</Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
