import { getBackendUrl } from "@/lib/utils";
import { getAuthHeadersFromContext } from "@/lib/auth";
import { fetchOidcUser } from "@/lib/oidcUser";

const DEFAULT_USER_ID = "admin@kagent.dev";
const USER_ID_KEY = "kagent_user_id";

const USER_ID_HEADER_CANDIDATES = [
  "x-user-id",
  "x-auth-request-email",
  "x-forwarded-email",
  "x-forwarded-user",
  "x-auth-request-user",
];

function getUserIdFromHeaders(reqHeaders: Headers): string | null {
  for (const headerName of USER_ID_HEADER_CANDIDATES) {
    const headerValue = reqHeaders.get(headerName)?.trim();
    if (headerValue) {
      return headerValue;
    }
  }

  return null;
}

export async function getCurrentUserId() {
  // Server-side path (server actions): read identity directly from request headers.
  if (typeof window === "undefined") {
    try {
      const { headers } = await import("next/headers");
      const reqHeaders = await headers();
      const headerUserId = getUserIdFromHeaders(reqHeaders);

      if (headerUserId) {
        return headerUserId;
      }
    } catch {
      // Ignore and continue to fallback behavior below.
    }
  }

  // Also check user id cached by localStorage.
  if (typeof window !== "undefined") {
    const storedUserId = localStorage.getItem(USER_ID_KEY);
    if (storedUserId && storedUserId !== DEFAULT_USER_ID) {
      return storedUserId;
    }
  }

  try {
    const user = await fetchOidcUser();
    if (user?.email) {
      if (typeof window !== "undefined") {
        localStorage.setItem(USER_ID_KEY, user.email);
      }
      return user.email;
    }
  } catch {
    // Ignore and continue to fallback behavior below.
  }

  return DEFAULT_USER_ID;
}

type ApiOptions = RequestInit & {
  method?: "GET" | "POST" | "PUT" | "DELETE" | "PATCH";
};

/**
 * Generic API fetch function with error handling
 * @param path API path to fetch
 * @param options Fetch options
 * @returns Promise with the response data
 * @throws Error with a descriptive message if the request fails
 */
export async function fetchApi<T>(path: string, options: ApiOptions = {}): Promise<T> {
  // Ensure path starts with a slash
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const url = `${getBackendUrl()}${cleanPath}`;

  // Get auth headers from incoming request (set by proxy)
  const authHeaders = await getAuthHeadersFromContext();

  try {
    const response = await fetch(url, {
      ...options,
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
        ...authHeaders,
        ...options.headers,
      },
      signal: AbortSignal.timeout(30000), // 30 second timeout
    });

    if (!response.ok) {
      // Try to extract error message from response
      let errorMessage = `Request failed with status ${response.status}. ${url}`;
      try {
        const contentType = response.headers.get("content-type");
        if (contentType && contentType.includes("application/json")) {
          const errorData = await response.json();
          if (errorData.error) {
            errorMessage = errorData.error;
          } else if (errorData.message) {
            errorMessage = errorData.message;
          }
        }
      } catch (parseError) {
        // If we can't parse the error response, use the default error message
        console.warn("Could not parse error response:", parseError);
      }

      throw new Error(errorMessage);
    }

    // Handle 204 No Content response (common for DELETE)
    if (response.status === 204) {
      return {} as T;
    }

    const contentType = response.headers.get("content-type");
    if (!contentType || !contentType.includes("application/json")) {
      throw new Error("Response was not JSON");
    }

    const jsonResponse = await response.json();
    return jsonResponse;
  } catch (error) {
    if (error instanceof TypeError && error.message === "Failed to fetch") {
      throw new Error(`Network error - Could not reach backend server. ${url}`);
    }
    if (error instanceof DOMException && error.name === "AbortError") {
      throw new Error(`Request timed out - server took too long to respond. ${url}`);
    }

    console.error("Error in fetchApi:", {
      path,
      url: `${path}`,
      error: error instanceof Error ? error.message : "Unknown error",
    });

    // Include more error details for debugging
    throw new Error(`${error instanceof Error ? error.message : "Unknown error"}`);
  }
}

/**
 * Helper function to create a standardized error response
 * @param error The error object
 * @param defaultMessage Default error message if the error doesn't have a message
 * @returns A BaseResponse object with error information
 */
export function createErrorResponse<T>(error: unknown, defaultMessage: string): { message: string; data?: T } {
  const errorMessage = error instanceof Error ? error.message : defaultMessage;
  console.error(defaultMessage, error);
  return { message: errorMessage };
}
