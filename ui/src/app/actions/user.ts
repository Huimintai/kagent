"use server";

import { getCurrentUserId } from "./utils";

/**
 * Server Action to get the current user's ID.
 * Used by client components to initialize user identity from server-side headers.
 * In production (behind oauth2-proxy), returns the user's email from X-Auth-Request-Email.
 * In local development, falls back to "admin@kagent.dev".
 */
export async function getServerUserId(): Promise<string> {
  return getCurrentUserId();
}
