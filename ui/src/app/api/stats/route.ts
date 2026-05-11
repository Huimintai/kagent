import { NextRequest, NextResponse } from "next/server";
import { getBackendUrl } from "@/lib/utils";
import { getAuthHeadersFromContext } from "@/lib/auth";

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;
  const limit = searchParams.get("limit") || "10";
  const excludePattern = process.env.KAGENT_DASHBOARD_EXCLUDE_PATTERN || "";

  const params = new URLSearchParams({ limit });
  if (excludePattern) {
    params.set("excludePattern", excludePattern);
  }

  const authHeaders = await getAuthHeadersFromContext();
  const res = await fetch(`${getBackendUrl()}/stats?${params.toString()}`, {
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...authHeaders },
  });
  const data = await res.json();
  return NextResponse.json(data, { status: res.status });
}
