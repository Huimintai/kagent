import { NextRequest, NextResponse } from 'next/server';
import crypto from 'crypto';
import { getGitHubInstances, getGitHubInstanceById, githubApiBase, revokeGitHubGrant, tokenCookie } from '@/lib/github';

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const instanceId = searchParams.get('instance');
  const publicUrl = process.env.GITHUB_OAUTH_REDIRECT_ORIGIN || `http://${req.headers.get('host') || 'localhost:8082'}`;

  // Resolve the requested instance
  const instances = getGitHubInstances();
  const instance = instanceId ? getGitHubInstanceById(instanceId) : instances[0];

  if (!instance) {
    return NextResponse.json({ error: 'GitHub OAuth not configured or unknown instance' }, { status: 400 });
  }

  // Revoke existing grant for this instance if a token exists
  const existingToken = req.cookies.get(tokenCookie(instance.id))?.value;
  if (existingToken) {
    const apiBase = githubApiBase(instance.oauthUrl);
    await revokeGitHubGrant(apiBase, instance.clientId, instance.clientSecret, existingToken);
  }

  // Generate CSRF state token
  const state = crypto.randomBytes(16).toString('hex');
  const referer = req.headers.get('referer') || '/';

  // Build the GitHub authorization URL
  const redirectUri = `${publicUrl}/actions/api/auth/github/callback`;
  const params = new URLSearchParams({
    client_id: instance.clientId,
    redirect_uri: redirectUri,
    scope: instance.scope,
    state,
  });

  const response = NextResponse.redirect(`${instance.oauthUrl}?${params.toString()}`);

  // Store transient cookies for the callback
  response.cookies.set('kagent_oauth_state', state, {
    httpOnly: true, sameSite: 'lax', path: '/', maxAge: 600,
  });
  response.cookies.set('kagent_oauth_referer', referer, {
    httpOnly: true, sameSite: 'lax', path: '/', maxAge: 600,
  });
  response.cookies.set('kagent_oauth_instance', instance.id, {
    httpOnly: true, sameSite: 'lax', path: '/', maxAge: 600,
  });

  return response;
}
