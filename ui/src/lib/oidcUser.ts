// Get user info from same origin API route that proxies OIDC headers from the ingress
export async function fetchOidcUser(): Promise<{ name: string, email: string } | null> {
  try {
    const res = await fetch(`/api/auth/user`, {
      credentials: "include",
    });
    if (!res.ok) return null;
    return await res.json();
  } catch {
    return null;
  }
}
