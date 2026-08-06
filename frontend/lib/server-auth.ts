import { cookies } from "next/headers";
import { API_BASE_URL } from "@/lib/api";
import type { AuthUser, Workspace } from "@/lib/auth-api";

/**
 * Server-only helpers for reading the authenticated session inside Server
 * Components (route guards). They forward the incoming session cookie to the Go
 * backend, so protection happens on the server before any UI renders — no auth
 * flash, and the httpOnly cookie is never exposed to client JS.
 */

async function serverFetch<T>(path: string): Promise<T | null> {
  const cookieHeader = (await cookies()).toString();
  try {
    const res = await fetch(`${API_BASE_URL}/api/v1${path}`, {
      headers: { Cookie: cookieHeader },
      cache: "no-store",
    });
    if (!res.ok) return null;
    const body = (await res.json()) as
      { success: true; data: T } | { success: false };
    return body.success ? body.data : null;
  } catch {
    return null; // backend unreachable → treat as unauthenticated
  }
}

/** The current user, or null if not signed in. */
export async function getServerUser(): Promise<AuthUser | null> {
  const data = await serverFetch<{ user: AuthUser }>("/auth/me");
  return data?.user ?? null;
}

/** The workspaces the current user belongs to (empty if none / not signed in). */
export async function getServerWorkspaces(): Promise<Workspace[]> {
  const data = await serverFetch<{ workspaces: Workspace[] }>("/workspaces");
  return data?.workspaces ?? [];
}
