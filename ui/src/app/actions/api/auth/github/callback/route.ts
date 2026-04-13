import { NextRequest, NextResponse } from 'next/server';
import { getGitHubInstanceById, tokenCookie, connectedCookie } from '@/lib/github';

export async function GET(req: NextRequest) {
  const { searchParams } = new URL(req.url);
  const code = searchParams.get('code');
  const state = searchParams.get('state');

  if (!code) {
    return NextResponse.json({ error: 'Missing authorization code' }, { status: 400 });
  }

  // CSRF check
  const savedState = req.cookies.get('kagent_oauth_state')?.value;
  if (!state || !savedState || state !== savedState) {
    return NextResponse.json({ error: 'Invalid OAuth state' }, { status: 403 });
  }

  // Resolve which instance this callback belongs to
  const instanceId = req.cookies.get('kagent_oauth_instance')?.value || 'default';
  const instance = getGitHubInstanceById(instanceId);
  if (!instance) {
    return NextResponse.json({ error: `Unknown GitHub instance: ${instanceId}` }, { status: 400 });
  }

  const publicUrl = process.env.GITHUB_OAUTH_REDIRECT_ORIGIN || `http://${req.headers.get('host') || 'localhost:8082'}`;

  // Exchange code for access token
  const redirectUri = `${publicUrl}/actions/api/auth/github/callback`;
  const tokenResp = await fetch(instance.tokenUrl, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    },
    body: JSON.stringify({
      client_id: instance.clientId,
      client_secret: instance.clientSecret,
      code,
      redirect_uri: redirectUri,
    }),
  });

  if (!tokenResp.ok) {
    const text = await tokenResp.text();
    return NextResponse.json({ error: `Token exchange failed: ${text}` }, { status: 502 });
  }

  const tokenData = await tokenResp.json();
  const accessToken = tokenData.access_token;

  if (!accessToken) {
    return NextResponse.json({ error: `No access_token in response: ${JSON.stringify(tokenData)}` }, { status: 502 });
  }

  // Redirect back to where the user was
  const referer = req.cookies.get('kagent_oauth_referer')?.value || '/';
  const response = NextResponse.redirect(referer);

  // Per-instance cookies
  response.cookies.set(tokenCookie(instance.id), accessToken, {
    httpOnly: true, sameSite: 'lax', path: '/', maxAge: 86400,
  });
  response.cookies.set(connectedCookie(instance.id), 'true', {
    httpOnly: false, sameSite: 'lax', path: '/', maxAge: 86400,
  });

  // Clean up transient cookies
  response.cookies.delete('kagent_oauth_state');
  response.cookies.delete('kagent_oauth_referer');
  response.cookies.delete('kagent_oauth_instance');

  return response;
}
