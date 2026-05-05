import { NextRequest, NextResponse } from 'next/server';
import { getGitHubInstanceById, githubApiBase, revokeGitHubGrant, tokenCookie, connectedCookie } from '@/lib/github';

export async function POST(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const instanceId = searchParams.get('instance');

  if (!instanceId) {
    return NextResponse.json({ error: 'Missing instance parameter' }, { status: 400 });
  }

  const instance = getGitHubInstanceById(instanceId);
  if (!instance) {
    return NextResponse.json({ error: `Unknown instance: ${instanceId}` }, { status: 400 });
  }

  const accessToken = req.cookies.get(tokenCookie(instanceId))?.value;

  // Revoke the OAuth app grant
  if (accessToken) {
    const apiBase = githubApiBase(instance.oauthUrl);
    await revokeGitHubGrant(apiBase, instance.clientId, instance.clientSecret, accessToken);
  }

  const response = NextResponse.json({ disconnected: true });

  response.cookies.set(tokenCookie(instanceId), '', {
    httpOnly: true, sameSite: 'lax', path: '/', maxAge: 0,
  });
  response.cookies.set(connectedCookie(instanceId), '', {
    httpOnly: false, sameSite: 'lax', path: '/', maxAge: 0,
  });

  return response;
}
