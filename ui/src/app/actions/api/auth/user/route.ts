import { NextRequest, NextResponse } from 'next/server';

export async function GET(req: NextRequest) {
  // Try common OIDC proxy headers for name
  const name = req.headers.get('x-forwarded-user')
    || req.headers.get('x-forwarded-preferred-username')
    || req.headers.get('x-auth-request-user');

  // Try common OIDC proxy headers for email
  const email = req.headers.get('x-auth-request-email')
    || req.headers.get('x-forwarded-email');

  return NextResponse.json({
    name: name || 'Unknown',
    email: email || null,
  });
}
