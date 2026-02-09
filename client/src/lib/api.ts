import { API_BASE_URL } from "./constants";
import type { CreateTaskRequest, CreateTaskResponse, Task, ApiError } from "@/types";

export class ApiRequestError extends Error {
  public readonly status: number;
  public readonly statusText: string;

  constructor(message: string, status: number, statusText: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
    this.statusText = statusText;
  }
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${API_BASE_URL}${path}`;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });

  if (!response.ok) {
    let errorMessage = `Request failed: ${response.status} ${response.statusText}`;
    try {
      const body: ApiError = await response.json();
      if (body.error) {
        errorMessage = body.error;
      }
    } catch {
      // Response body was not JSON
    }
    throw new ApiRequestError(errorMessage, response.status, response.statusText);
  }

  return response.json() as Promise<T>;
}

export async function createTask(prompt: string): Promise<CreateTaskResponse> {
  const body: CreateTaskRequest = { prompt };
  return apiFetch<CreateTaskResponse>("/api/task", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export async function getTask(taskId: string): Promise<Task> {
  return apiFetch<Task>(`/api/task/${encodeURIComponent(taskId)}`);
}
