import { notifications } from "@mantine/notifications";

import type { ApiError } from "./types";

// Base path is empty: in dev the Vite proxy forwards /api to :8081, in prod
// the Go binary serves both the assets and the API on the same origin.
const BASE = "";

export class HttpError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "HttpError";
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (res.status === 204) {
    return undefined as T;
  }

  const text = await res.text();
  let json: unknown = undefined;
  if (text) {
    try {
      json = JSON.parse(text);
    } catch {
      // non-JSON body (eg. plain "OK")
      json = text;
    }
  }

  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    if (json && typeof json === "object" && typeof (json as ApiError).error === "string") {
      msg = (json as ApiError).error;
    } else if (typeof json === "string" && json.length > 0) {
      msg = json;
    }
    notifications.show({ color: "red", title: `Request failed (${res.status})`, message: msg });
    throw new HttpError(res.status, msg);
  }

  return json as T;
}

export const http = {
  get: <T>(path: string) => request<T>("GET", path),
  post: <T>(path: string, body?: unknown) => request<T>("POST", path, body),
  patch: <T>(path: string, body?: unknown) => request<T>("PATCH", path, body),
  del: <T>(path: string) => request<T>("DELETE", path),
};
