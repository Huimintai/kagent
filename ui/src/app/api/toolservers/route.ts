import { NextResponse } from "next/server";
import { getBackendUrl } from "@/lib/utils";
import { getAuthHeadersFromContext } from "@/lib/auth";

export async function GET() {
  const authHeaders = await getAuthHeadersFromContext();
  const res = await fetch(`${getBackendUrl()}/toolservers`, {
    cache: "no-store",
    headers: { "Content-Type": "application/json", ...authHeaders },
  });
  const data = await res.json();
  return NextResponse.json(data, { status: res.status });
}
