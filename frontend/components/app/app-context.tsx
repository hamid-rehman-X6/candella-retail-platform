"use client";

import { createContext, useContext, useState } from "react";
import type { AuthUser, Workspace } from "@/lib/auth-api";

type AppContextValue = {
  user: AuthUser;
  workspaces: Workspace[];
  currentWorkspace: Workspace;
  setCurrentWorkspaceId: (id: string) => void;
};

const AppContext = createContext<AppContextValue | null>(null);

/**
 * Holds the signed-in user and their workspaces for the authenticated app.
 * Seeded from the server (the dashboard layout), so there's no client refetch on
 * first paint. The dashboard layout guarantees at least one workspace exists.
 */
export function AppProvider({
  user,
  workspaces,
  children,
}: {
  user: AuthUser;
  workspaces: Workspace[];
  children: React.ReactNode;
}) {
  const [currentId, setCurrentId] = useState(workspaces[0]?.id ?? "");
  // Non-null: the layout redirects to onboarding when there are no workspaces.
  const currentWorkspace = (workspaces.find((w) => w.id === currentId) ??
    workspaces[0])!;

  return (
    <AppContext.Provider
      value={{
        user,
        workspaces,
        currentWorkspace,
        setCurrentWorkspaceId: setCurrentId,
      }}
    >
      {children}
    </AppContext.Provider>
  );
}

export function useApp(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useApp must be used within <AppProvider>");
  return ctx;
}
