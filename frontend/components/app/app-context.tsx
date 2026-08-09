"use client";

import { createContext, useContext } from "react";
import type { AuthUser, Workspace } from "@/lib/auth-api";

type AppContextValue = {
  user: AuthUser;
  workspaces: Workspace[];
  /** The workspace this page is for, resolved from the URL by the layout. */
  current: Workspace;
};

const AppContext = createContext<AppContextValue | null>(null);

/**
 * Holds the signed-in user, their workspaces, and the one currently addressed.
 * All three are seeded from the server (the workspace layout), so there's no
 * client refetch on first paint — and `current` comes from the URL rather than
 * client state, which is what keeps two tabs on two workspaces from colliding.
 */
export function AppProvider({
  user,
  workspaces,
  current,
  children,
}: {
  user: AuthUser;
  workspaces: Workspace[];
  current: Workspace;
  children: React.ReactNode;
}) {
  return (
    <AppContext.Provider value={{ user, workspaces, current }}>
      {children}
    </AppContext.Provider>
  );
}

export function useApp(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used within <AppProvider>");
  return ctx;
}

/** Builds a URL inside a workspace, e.g. workspacePath("acme", "/products"). */
export function workspacePath(slug: string, path = ""): string {
  return `/w/${slug}${path}`;
}
