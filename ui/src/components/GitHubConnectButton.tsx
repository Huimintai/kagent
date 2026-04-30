"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ChevronDown, ExternalLink } from "lucide-react";

interface InstanceStatus {
  id: string;
  label: string;
  connected: boolean;
  disabled?: boolean;
  loginUrl?: string;
}

function getCookie(name: string): string | null {
  if (typeof document === "undefined") return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

const GitHubIcon = () => (
  <svg viewBox="0 0 16 16" width="14" height="14" fill="currentColor">
    <path d="M8 0c4.42 0 8 3.58 8 8a8.013 8.013 0 0 1-5.45 7.59c-.4.08-.55-.17-.55-.38 0-.27.01-1.13.01-2.2 0-.75-.25-1.23-.54-1.48 1.78-.2 3.65-.88 3.65-3.95 0-.88-.31-1.59-.82-2.15.08-.2.36-1.02-.08-2.12 0 0-.67-.22-2.2.82-.64-.18-1.32-.27-2-.27-.68 0-1.36.09-2 .27-1.53-1.03-2.2-.82-2.2-.82-.44 1.1-.16 1.92-.08 2.12-.51.56-.82 1.28-.82 2.15 0 3.06 1.86 3.75 3.64 3.95-.23.2-.44.55-.51 1.07-.46.21-1.61.55-2.33-.66-.15-.24-.6-.83-1.23-.82-.67.01-.27.38.01.53.34.19.73.9.82 1.13.16.45.68 1.31 2.69.94 0 .67.01 1.3.01 1.49 0 .21-.15.45-.55.38A7.995 7.995 0 0 1 0 8c0-4.42 3.58-8 8-8Z" />
  </svg>
);

// Smoke-test flag — flip locally to verify the GitHub expiry banner without waiting for a real token expiry.
// DEV_GITHUB_EXPIRE_ON_LOAD: treat all connected instances as expired immediately on page load.
// Defaults to false — never commit with this enabled.
const DEV_GITHUB_EXPIRE_ON_LOAD = false;

interface GitHubConnectButtonProps {
  onTokenExpired?: (labels: string[]) => void;
  onDropdownOpen?: () => void;
}

export default function GitHubConnectButton({ onTokenExpired, onDropdownOpen }: GitHubConnectButtonProps) {
  const [instances, setInstances] = useState<InstanceStatus[]>([]);
  const [disconnecting, setDisconnecting] = useState<InstanceStatus | null>(null);
  const [connecting, setConnecting] = useState<InstanceStatus | null>(null);

  useEffect(() => {
    fetch("/actions/api/auth/github/status")
      .then((r) => r.json())
      .then((data) => {
        if (!data.instances) return;
        const fetched: InstanceStatus[] = data.instances;
        setInstances(fetched);

        if (DEV_GITHUB_EXPIRE_ON_LOAD) {
          const expiredLabels = fetched.filter((i) => !i.disabled).map((i) => i.label);
          if (expiredLabels.length === 0) return;
          onTokenExpired?.(expiredLabels);
          return;
        }

        // Validate tokens for connected, non-disabled instances against the GitHub API.
        // A present cookie does not guarantee the token is still accepted.
        // Disabled instances are unavailable by config — skip expiry checks for them.
        const connected = fetched.filter((i) => i.connected && !i.disabled);
        if (connected.length === 0) return;

        fetch("/actions/api/auth/github/validate")
          .then((r) => r.json())
          .then((v: { instances: { id: string; valid: boolean }[] }) => {
            const invalidIds = new Set(
              v.instances.filter((r) => !r.valid).map((r) => r.id)
            );
            if (invalidIds.size === 0) return;

            setInstances((prev) =>
              prev.map((i) => invalidIds.has(i.id) ? { ...i, connected: false } : i)
            );
            const expiredLabels = fetched
              .filter((i) => invalidIds.has(i.id) && !i.disabled)
              .map((i) => i.label);
            onTokenExpired?.(expiredLabels);
          })
          .catch(() => {/* best-effort — don't disrupt UI on network error */});
      })
      .catch(() => setInstances([]));
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  // Client-side cookie sync (picks up changes after OAuth redirect)
  useEffect(() => {
    setInstances((prev) =>
      prev.map((inst) => ({
        ...inst,
        connected: inst.connected || getCookie(`kagent_github_connected_${inst.id}`) === "true",
      }))
    );
  }, []);

  if (instances.length === 0) return null;

  const connectedInstances = instances.filter((i) => i.connected);
  const disconnectedInstances = instances.filter((i) => !i.connected);

  const handleConnect = (inst: InstanceStatus) => {
    if (inst.loginUrl) {
      setConnecting(inst);
    } else {
      window.location.href = `/actions/api/auth/github?instance=${encodeURIComponent(inst.id)}`;
    }
  };

  const handleProceedConnect = (inst: InstanceStatus) => {
    setConnecting(null);
    window.location.href = `/actions/api/auth/github?instance=${encodeURIComponent(inst.id)}`;
  };

  const handleDisconnect = (inst: InstanceStatus) => {
    fetch(`/actions/api/auth/github/disconnect?instance=${encodeURIComponent(inst.id)}`, { method: "POST" })
      .then(() => {
        setInstances((prev) => prev.map((i) => (i.id === inst.id ? { ...i, connected: false } : i)));
      })
      .catch(() => {
        setInstances((prev) => prev.map((i) => (i.id === inst.id ? { ...i, connected: false } : i)));
      })
      .finally(() => setDisconnecting(null));
  };

  // --- Single instance: simple button ---
  if (instances.length === 1) {
    const inst = instances[0];
    if (inst.connected) {
      return (
        <>
          <Button
            variant="outline"
            size="sm"
            className="text-xs gap-1.5 text-green-600 border-green-200 hover:border-red-200 hover:text-red-600 group"
            onClick={() => setDisconnecting(inst)}
          >
            <GitHubIcon />
            <span className="group-hover:hidden">{inst.label}</span>
            <span className="hidden group-hover:inline">Disconnect</span>
          </Button>
          <DisconnectDialog inst={disconnecting} onCancel={() => setDisconnecting(null)} onConfirm={handleDisconnect} />
        </>
      );
    }
    return (
      <>
        <Button variant="outline" size="sm" className="text-xs gap-1.5" onClick={() => { onDropdownOpen?.(); handleConnect(inst); }}>
          <GitHubIcon />
          Connect GitHub
        </Button>
        <PreLoginDialog inst={connecting} onCancel={() => setConnecting(null)} onConfirm={handleProceedConnect} />
      </>
    );
  }

  // --- Multiple instances: dropdown ---
  return (
    <>
      <DropdownMenu onOpenChange={(open) => { if (open) onDropdownOpen?.(); }}>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="text-xs gap-1.5">
            <GitHubIcon />
            GitHub
            {connectedInstances.length > 0 && (
              <span className="text-green-600 ml-0.5">({connectedInstances.length})</span>
            )}
            <ChevronDown className="h-3 w-3" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-56">
          {connectedInstances.length > 0 && (
            <>
              {connectedInstances.map((inst) => (
                <DropdownMenuItem
                  key={inst.id}
                  onClick={() => setDisconnecting(inst)}
                  className="cursor-pointer text-green-600 hover:!text-red-600"
                >
                  <span className="flex-1">{inst.label}</span>
                  <span className="text-xs opacity-60">Disconnect</span>
                </DropdownMenuItem>
              ))}
              {disconnectedInstances.length > 0 && <DropdownMenuSeparator />}
            </>
          )}
          {disconnectedInstances.map((inst) => (
            <DropdownMenuItem
              key={inst.id}
              onClick={() => !inst.disabled && handleConnect(inst)}
              className={inst.disabled ? "opacity-40 cursor-not-allowed" : "cursor-pointer"}
              disabled={inst.disabled}
            >
              <span className="flex-1">{inst.label}</span>
              <span className="text-xs opacity-60">{inst.disabled ? "Unavailable" : "Connect"}</span>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      <DisconnectDialog inst={disconnecting} onCancel={() => setDisconnecting(null)} onConfirm={handleDisconnect} />
      <PreLoginDialog inst={connecting} onCancel={() => setConnecting(null)} onConfirm={handleProceedConnect} />
    </>
  );
}

function DisconnectDialog({
  inst,
  onCancel,
  onConfirm,
}: {
  inst: InstanceStatus | null;
  onCancel: () => void;
  onConfirm: (inst: InstanceStatus) => void;
}) {
  return (
    <AlertDialog open={!!inst} onOpenChange={(open) => { if (!open) onCancel(); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Disconnect GitHub?</AlertDialogTitle>
          <AlertDialogDescription>
            This will revoke access to <strong>{inst?.label}</strong>. You will need to re-authorize to connect again.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={() => inst && onConfirm(inst)}
            className="bg-red-600 hover:bg-red-700"
          >
            Disconnect
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function PreLoginDialog({
  inst,
  onCancel,
  onConfirm,
}: {
  inst: InstanceStatus | null;
  onCancel: () => void;
  onConfirm: (inst: InstanceStatus) => void;
}) {
  return (
    <AlertDialog open={!!inst} onOpenChange={(open) => { if (!open) onCancel(); }}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Sign in to {inst?.label}</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-3">
              <p>
                To avoid being redirected away, please sign in to <strong>{inst?.label}</strong> first,
                then come back and click <strong>Continue</strong>.
              </p>
              {inst?.loginUrl && (
                <a
                  href={inst.loginUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1.5 text-sm font-medium text-blue-600 hover:text-blue-700 underline"
                >
                  Open {inst.label}
                  <ExternalLink className="h-3.5 w-3.5" />
                </a>
              )}
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={() => inst && onConfirm(inst)}>
            Continue
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
