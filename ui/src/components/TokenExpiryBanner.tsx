'use client';

import { useState } from "react";
import { AlertTriangle, RefreshCw, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/contexts/AuthContext";
import { useUserStore } from "@/lib/userStore";

export function TokenExpiryBanner() {
  const { tokenExpired } = useAuth();
  const renewToken = useUserStore((state) => state.renewToken);
  const [dismissed, setDismissed] = useState(false);

  if (!tokenExpired || dismissed) return null;

  return (
    <div className="absolute inset-x-0 top-4 md:top-8 z-50 flex justify-center px-4 md:px-6">
      <div className="w-full max-w-6xl bg-yellow-50 dark:bg-yellow-950 border border-yellow-500/30 rounded-md px-4 py-2 flex items-center justify-between text-sm">
        <span className="flex items-center gap-2 text-yellow-700 dark:text-yellow-400">
          <AlertTriangle className="h-4 w-4 shrink-0" />
          Your session has expired. Renew your OIDC login token to continue using this platform.
        </span>
        <div className="flex items-center gap-2 ml-4 shrink-0">
          <Button
            size="sm"
            variant="outline"
            className="gap-1 border-yellow-500/50 text-yellow-700 dark:text-yellow-400 hover:bg-yellow-500/10"
            onClick={renewToken}
          >
            <RefreshCw className="h-3 w-3" />
            Renew Token
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className="text-yellow-700 dark:text-yellow-400 hover:bg-yellow-500/10 p-1"
            onClick={() => setDismissed(true)}
            aria-label="Dismiss"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
