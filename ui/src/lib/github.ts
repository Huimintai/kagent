export interface GitHubInstance {
  id: string;
  label: string;
  clientId: string;
  clientSecret: string;
  oauthUrl: string;
  tokenUrl: string;
  scope: string;
  disabled?: boolean;
}

/**
 * Load configured GitHub instances.
 * Checks GITHUB_INSTANCES JSON env var first, falls back to legacy individual env vars.
 */
export function getGitHubInstances(): GitHubInstance[] {
  const raw = process.env.GITHUB_INSTANCES;
  if (raw) {
    try {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed) && parsed.length > 0) {
        return parsed.map((inst: Record<string, unknown>) => ({
          id: inst.id as string,
          label: (inst.label || inst.id) as string,
          clientId: inst.clientId as string,
          clientSecret: inst.clientSecret as string,
          oauthUrl: inst.oauthUrl as string,
          tokenUrl: inst.tokenUrl as string,
          scope: (inst.scope as string) || 'repo read:org read:user',
          disabled: !!inst.disabled,
        }));
      }
    } catch {
      // fall through to legacy
    }
  }

  // Legacy single-instance env vars
  const clientId = process.env.GITHUB_CLIENT_ID;
  if (!clientId) return [];

  const oauthUrl = process.env.GITHUB_OAUTH_URL || 'https://github.com/login/oauth/authorize';
  let label: string;
  try {
    label = new URL(oauthUrl).hostname;
  } catch {
    label = 'GitHub';
  }

  return [{
    id: 'default',
    label,
    clientId,
    clientSecret: process.env.GITHUB_CLIENT_SECRET || '',
    oauthUrl,
    tokenUrl: process.env.GITHUB_TOKEN_URL || 'https://github.com/login/oauth/access_token',
    scope: process.env.GITHUB_OAUTH_SCOPE || 'repo read:org read:user',
  }];
}

// Per-instance cookie names
export function tokenCookie(id: string): string {
  return `kagent_github_token_${id}`;
}
export function connectedCookie(id: string): string {
  return `kagent_github_connected_${id}`;
}

export function getGitHubInstanceById(id: string): GitHubInstance | undefined {
  return getGitHubInstances().find((inst) => inst.id === id);
}

/**
 * Derive GitHub API base URL from the OAuth URL.
 * github.com  → https://api.github.com
 * GHE         → https://<host>/api/v3
 */
export function githubApiBase(oauthUrl: string): string {
  const url = new URL(oauthUrl);
  if (url.hostname === 'github.com') {
    return 'https://api.github.com';
  }
  return `${url.origin}/api/v3`;
}

/**
 * Revoke the entire OAuth app grant so GitHub shows the authorization prompt again.
 */
export async function revokeGitHubGrant(
  apiBase: string,
  clientId: string,
  clientSecret: string,
  accessToken: string,
): Promise<void> {
  try {
    await fetch(`${apiBase}/applications/${clientId}/grant`, {
      method: 'DELETE',
      headers: {
        'Authorization': 'Basic ' + Buffer.from(`${clientId}:${clientSecret}`).toString('base64'),
        'Accept': 'application/vnd.github+json',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ access_token: accessToken }),
    });
  } catch {
    // Best-effort
  }
}
