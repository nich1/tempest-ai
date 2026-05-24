import type { ApiError, Job, User } from "./types";

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers || {}),
    },
  });
  if (!res.ok) {
    let body: ApiError = { error: res.statusText };
    try {
      body = await res.json();
    } catch {
      // body already populated with statusText
    }
    throw new Error(body.error || res.statusText);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  signup: (email: string, password: string) =>
    request<{ user: User }>("/auth/signup", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  login: (email: string, password: string) =>
    request<{ user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),
  logout: () => request<void>("/auth/logout", { method: "POST" }),
  me: () => request<User>("/auth/me"),
  listJobs: () =>
    request<{ jobs: Job[]; total_count: number }>("/jobs"),
  getJob: (id: string) => request<Job>(`/jobs/${id}`),
  createJob: (body: unknown) =>
    request<Job>("/jobs", { method: "POST", body: JSON.stringify(body) }),
  fileUploadURL: (content_type: string, size_bytes: number) =>
    request<{
      upload_url: string;
      blob_key: string;
      max_size_bytes: number;
      expires_in_seconds: number;
    }>("/jobs/file-upload-url", {
      method: "POST",
      body: JSON.stringify({ content_type, size_bytes }),
    }),
};
