"use server";

import { BaseResponse, StatsResponse } from "@/types";
import { fetchApi, createErrorResponse } from "./utils";

export async function getStats(limit: number = 10): Promise<BaseResponse<StatsResponse>> {
  try {
    const data = await fetchApi<BaseResponse<StatsResponse>>(`/stats?limit=${limit}`);
    return data;
  } catch (error) {
    return createErrorResponse<StatsResponse>(error, "Failed to load stats");
  }
}
