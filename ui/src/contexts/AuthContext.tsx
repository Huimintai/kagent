"use client";

import React, { createContext, useContext, useEffect, useState, useCallback, useRef, ReactNode } from "react";
import { getCurrentUser, CurrentUser } from "@/app/actions/auth";

interface AuthContextValue {
  user: CurrentUser | null;
  isLoading: boolean;
  error: Error | null;
  tokenExpired: boolean;
  refetch: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const TOKEN_CHECK_INTERVAL_MS = 60_000;

// Smoke-test flags — flip these locally to verify the banner without waiting for a real expiry.
// DEV_EXPIRE_ON_VISIBILITY: treat token as expired every time the tab regains focus.
// DEV_EXPIRE_AFTER_MS: treat token as expired after this many ms (e.g. 2 * 60_000 for 2 min).
// Both default to false/0 (disabled) — never commit with these enabled.
const DEV_EXPIRE_ON_VISIBILITY = false;
const DEV_EXPIRE_AFTER_MS = 0; // set to e.g. 2 * 60_000 to test 2-minute expiry

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const [tokenExpired, setTokenExpired] = useState(false);
  const wasAuthenticatedRef = useRef(false);
  const mountTimeRef = useRef(Date.now());
  const tokenExpiredRef = useRef(false);

  const fetchUser = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const currentUser = await getCurrentUser();
      setUser(currentUser);
      if (currentUser) wasAuthenticatedRef.current = true;
    } catch (e) {
      setError(e instanceof Error ? e : new Error("Failed to fetch user"));
    } finally {
      setIsLoading(false);
    }
  };

  const checkExpiry = useCallback(async () => {
    if (!wasAuthenticatedRef.current || tokenExpiredRef.current) return;
    if (DEV_EXPIRE_AFTER_MS > 0 && Date.now() - mountTimeRef.current >= DEV_EXPIRE_AFTER_MS) {
      tokenExpiredRef.current = true;
      setTokenExpired(true);
      return;
    }
    // The JWT bearer token is injected server-side by oauth2-proxy and is never
    // accessible to browser JS. Expiry is detected implicitly: oauth2-proxy stops
    // injecting auth headers once the session/token expires, so getCurrentUser()
    // returns null — no user means the token has expired.
    const currentUser = await getCurrentUser();
    if (!currentUser) {
      tokenExpiredRef.current = true;
      setTokenExpired(true);
    }
  }, []);

  useEffect(() => {
    fetchUser();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const interval = setInterval(checkExpiry, TOKEN_CHECK_INTERVAL_MS);
    const onVisibilityChange = () => {
      if (document.visibilityState !== "visible") return;
      if (DEV_EXPIRE_ON_VISIBILITY) {
        setTokenExpired(true);
        return;
      }
      checkExpiry();
    };
    document.addEventListener("visibilitychange", onVisibilityChange);
    return () => {
      clearInterval(interval);
      document.removeEventListener("visibilitychange", onVisibilityChange);
    };
  }, [checkExpiry]);

  return (
    <AuthContext.Provider value={{ user, isLoading, error, tokenExpired, refetch: fetchUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
