import type { ApiResponse } from "@candella/types";

/**
 * Base URL of the Go backend. In the browser this must be an absolute URL so
 * the cross-origin cookie flow works; it's configured via NEXT_PUBLIC_API_URL.
 */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** A typed error carrying the backend's stable `error.code` for UI branching. */
export class ApiError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.code = code;
  }
}

/** Header naming the workspace a request acts within (see backend RequireTenant). */
export const TENANT_HEADER = "X-Tenant-Id";

type FetchOptions = RequestInit & {
  /**
   * The workspace this call belongs to. Passed explicitly on every tenant-scoped
   * call rather than read from a global: with ambient state, two tabs open on two
   * workspaces race over one value and writes land in the wrong tenant.
   */
  tenantId?: string;
};

/**
 * Calls the backend and unwraps the shared `{ success, data | error }` envelope.
 * Always sends cookies (`credentials: "include"`) so the httpOnly session cookie
 * is set and returned. Throws an ApiError on any failure.
 */
export async function apiFetch<T>(
  path: string,
  options: FetchOptions = {},
): Promise<T> {
  const { tenantId, ...init } = options;

  let res: Response;
  try {
    res = await fetch(`${API_BASE_URL}/api/v1${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(tenantId ? { [TENANT_HEADER]: tenantId } : {}),
        ...(init.headers ?? {}),
      },
    });
  } catch {
    throw new ApiError(
      "network_error",
      "Can't reach the server. Check your connection and try again.",
    );
  }

  let body: ApiResponse<T>;
  try {
    body = (await res.json()) as ApiResponse<T>;
  } catch {
    throw new ApiError(
      "invalid_response",
      "The server returned an unexpected response.",
    );
  }

  if (!body.success) {
    throw new ApiError(body.error.code, body.error.message);
  }
  return body.data;
}

export function apiPost<T>(
  path: string,
  data?: unknown,
  tenantId?: string,
): Promise<T> {
  return apiFetch<T>(path, {
    method: "POST",
    body: data !== undefined ? JSON.stringify(data) : undefined,
    tenantId,
  });
}

export function apiGet<T>(path: string, tenantId?: string): Promise<T> {
  return apiFetch<T>(path, { method: "GET", tenantId });
}
