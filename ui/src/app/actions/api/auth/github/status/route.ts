import { NextRequest, NextResponse } from 'next/server';
import { getGitHubInstances, tokenCookie } from '@/lib/github';

export async function GET(req: NextRequest) {
  const instances = getGitHubInstances();
  const enabled = instances.length > 0;

  const instanceStatuses = instances.map(({ id, label, disabled, oauthUrl }) => {
    let loginUrl: string | undefined;
    try {
      loginUrl = new URL(oauthUrl).origin;
    } catch {
      // skip
    }
    return {
      id,
      label,
      connected: !!req.cookies.get(tokenCookie(id))?.value,
      disabled: !!disabled,
      loginUrl,
    };
  });

  return NextResponse.json({
    enabled,
    instances: instanceStatuses,
  });
}
