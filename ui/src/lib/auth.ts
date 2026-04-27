import { headers } from "next/headers";
import { NextRequest } from "next/server";

/**
 * Headers to forward from the incoming request (set by oauth2-proxy / ingress)
 * to the backend controller. Order matters for user-id: the controller's
 * UnsecureAuthenticator checks X-User-Id first, so we map the most reliable
 * source (X-Auth-Request-Email) into that header.
 */
const FORWARDED_HEADERS = [
  "Authorization",
  "X-Auth-Request-User",
  "X-Auth-Request-Email",
  "X-Forwarded-Email",
  "X-Forwarded-User",
];

/**
 * Extract authentication headers from a headers-like object.
 * Common implementation used by both server actions and route handlers.
 */
function extractAuthHeaders(getHeader: (name: string) => string | null): Record<string, string> {
  const authHeaders: Record<string, string> = {};

  for (const name of FORWARDED_HEADERS) {
    const value = getHeader(name);
    if (value) {
      authHeaders[name] = value;
    }
  }

  // Ensure X-User-Id is set so the controller can identify the user
  // even when running in "unsecure" auth mode (no JWT parsing).
  if (!authHeaders["X-User-Id"]) {
    const userId =
      getHeader("X-Auth-Request-Email") ||
      getHeader("X-Forwarded-Email") ||
      getHeader("X-Auth-Request-User") ||
      getHeader("X-Forwarded-User");
    if (userId) {
      authHeaders["X-User-Id"] = userId;
    }
  }

  return authHeaders;
}

/**
 * Get authentication headers from incoming request (for route handlers).
 * These are set by oauth2-proxy or other auth proxies.
 */
export function getAuthHeadersFromRequest(request: NextRequest): Record<string, string> {
  return extractAuthHeaders((name) => request.headers.get(name));
}

/**
 * Get authentication headers from request context (for server actions).
 * These are set by oauth2-proxy or other auth proxies.
 */
export async function getAuthHeadersFromContext(): Promise<Record<string, string>> {
  const headersList = await headers();
  return extractAuthHeaders((name) => headersList.get(name));
}
