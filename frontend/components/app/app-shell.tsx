"use client";

import { useState } from "react";
import { AppProvider } from "@/components/app/app-context";
import { Sidebar } from "@/components/app/sidebar";
import { Topbar } from "@/components/app/topbar";
import type { AuthUser, Workspace } from "@/lib/auth-api";

/**
 * The authenticated application frame: fixed sidebar + sticky topbar + content.
 * Seeded with the user and workspaces from the server so nothing refetches on
 * first paint. Owns the mobile-sidebar open state.
 */
export function AppShell({
  user,
  workspaces,
  children,
}: {
  user: AuthUser;
  workspaces: Workspace[];
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <AppProvider user={user} workspaces={workspaces}>
      <div className="min-h-screen bg-canvas">
        <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
        <div className="lg:pl-64">
          <Topbar onMenuClick={() => setSidebarOpen(true)} />
          <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8">
            {children}
          </main>
        </div>
      </div>
    </AppProvider>
  );
}
