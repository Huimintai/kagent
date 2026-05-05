'use client';

import { useState, useEffect } from "react";
import { AlertTriangle, RefreshCw, X, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/contexts/AuthContext";
import { useUserStore } from "@/lib/userStore";

// Full-width overlay banner for OIDC session expiry.
export function OidcExpiryBanner() {
  const { tokenExpired } = useAuth();
  const renewToken = useUserStore((state) => state.renewToken);
  const [dismissed, setDismissed] = useState(false);

  if (!tokenExpired || dismissed) return null;

  return (
    <div className="absolute inset-0 z-50 px-4 md:px-6 flex flex-col justify-center pointer-events-none">
      <div className="max-w-6xl mx-auto w-full pointer-events-auto bg-yellow-50 dark:bg-yellow-950 border border-yellow-500/30 rounded-md px-4 py-2 flex items-center justify-between text-sm">
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

interface GithubExpiryBannerProps {
  githubExpiredLabels: string[];
  // Width of the right-controls div measured from Header — used to size the
  // invisible right spacer so the banner stops exactly where the GitHub button starts.
  rightSpacerWidth: number;
}

// Overlay banner for GitHub token expiry. Narrower than the OIDC banner —
// a right spacer of exactly rightSpacerWidth px keeps it clear of the GitHub button.
export function GithubExpiryBanner({ githubExpiredLabels, rightSpacerWidth }: GithubExpiryBannerProps) {
  const [dismissed, setDismissed] = useState(false);

  // Reset dismissed state when a new set of expired labels arrives.
  useEffect(() => {
    if (githubExpiredLabels.length > 0) setDismissed(false);
  }, [githubExpiredLabels]);

  if (githubExpiredLabels.length === 0 || dismissed) return null;

  const githubLabel = githubExpiredLabels.length === 1
    ? githubExpiredLabels[0]
    : githubExpiredLabels.join(", ");

  return (
    <div className="absolute inset-0 z-50 flex flex-col justify-center pointer-events-none">
      <div className="max-w-6xl mx-auto w-full px-4 md:px-6 flex items-center gap-2">
        <div className="pointer-events-auto bg-orange-50 dark:bg-orange-950 border border-orange-500/30 rounded-md px-4 py-2 flex items-center justify-between gap-3 text-sm min-w-0 flex-1">
          <span className="flex items-center gap-2 text-orange-700 dark:text-orange-400 min-w-0">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            <span className="truncate">GitHub connection expired for <strong>{githubLabel}</strong>. Please click the GitHub button to reconnect.</span>
          </span>
          <ArrowRight className="h-4 w-4 shrink-0 text-orange-700 dark:text-orange-400" />
        </div>
        {/* Invisible spacer matching the right-controls div width — keeps banner from overlapping the GitHub button */}
        <div style={{ width: rightSpacerWidth }} className="shrink-0" />
      </div>
    </div>
  );
}

// Keep TokenExpiryBanner as a re-export for backward compatibility.
export function TokenExpiryBanner() {
  return <OidcExpiryBanner />;
}
