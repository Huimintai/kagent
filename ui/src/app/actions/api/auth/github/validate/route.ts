import { NextRequest, NextResponse } from 'next/server';
import { getGitHubInstances, tokenCookie, githubApiBase } from '@/lib/github';

export async function GET(req: NextRequest) {
  const instances = getGitHubInstances();

  const results = await Promise.all(
    instances.map(async ({ id, oauthUrl }) => {
      const token = req.cookies.get(tokenCookie(id))?.value;
      if (!token) return { id, valid: false };

      try {
        // A lightweight authenticated call — just checks the token is still accepted.
        const resp = await fetch(`${githubApiBase(oauthUrl)}/user`, {
          headers: {
            Authorization: `Bearer ${token}`,
            Accept: 'application/vnd.github+json',
          },
        });
        return { id, valid: resp.ok };
      } catch {
        return { id, valid: false };
      }
    })
  );

  return NextResponse.json({ instances: results });
}
