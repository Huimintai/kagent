"use server";

import { AgentComment, BaseResponse } from "@/types";
import { fetchApi } from "./utils";

export async function getAgentComments(namespace: string, name: string, limit: number = 50) {
  try {
    const response = await fetchApi<BaseResponse<AgentComment[]>>(
      `/agents/${namespace}/${name}/comments?limit=${limit}`,
      { method: "GET" },
    );
    return { data: response.data ?? [], error: null };
  } catch (error) {
    return { data: null, error: error instanceof Error ? error.message : "Failed to load comments" };
  }
}

export async function createAgentComment(namespace: string, name: string, content: string) {
  try {
    const response = await fetchApi<BaseResponse<AgentComment>>(
      `/agents/${namespace}/${name}/comments`,
      {
        method: "POST",
        body: JSON.stringify({ content }),
      },
    );
    return { data: response.data ?? null, error: null };
  } catch (error) {
    return { data: null, error: error instanceof Error ? error.message : "Failed to post comment" };
  }
}

export async function deleteAgentComment(namespace: string, name: string, commentId: string) {
  try {
    await fetchApi<BaseResponse<unknown>>(
      `/agents/${namespace}/${name}/comments/${commentId}`,
      { method: "DELETE" },
    );
    return { error: null };
  } catch (error) {
    return { error: error instanceof Error ? error.message : "Failed to delete comment" };
  }
}
