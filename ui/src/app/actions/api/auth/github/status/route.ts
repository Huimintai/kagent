import { NextRequest, NextResponse } from 'next/server';
import { getGitHubInstances, tokenCookie, connectedCookie } from '@/lib/github';

export async function GET(req: NextRequest) {
  const instances = getGitHubInstances();
  const enabled = instances.length > 0;

  const instanceStatuses = instances.map(({ id, label, disabled }) => ({
    id,
    label,
    connected: !!req.cookies.get(tokenCookie(id))?.value,
    disabled: !!disabled,
  }));

  return NextResponse.json({
    enabled,
    instances: instanceStatuses,
  });
}
