"use client";

import { useState } from "react";
import { AppProvider } from "@/components/app/app-context";
import { Sidebar } from "@/components/app/sidebar";
import { Topbar } from "@/components/app/topbar";
import type { AuthUser, Workspace } from "@/lib/auth-api";

/**
 * The authenticated application frame: fixed sidebar + sticky topbar + content.
 * Seeded with the user and workspaces from the server so nothing refetches on
 * first paint. Owns the mobile drawer and desktop sidebar states.
 */
export function AppShell({
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
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <AppProvider user={user} workspaces={workspaces} current={current}>
      <div className="min-h-screen bg-canvas print:bg-white">
        <Sidebar
          open={sidebarOpen}
          onClose={() => setSidebarOpen(false)}
          collapsed={sidebarCollapsed}
          onToggleCollapsed={() => setSidebarCollapsed((value) => !value)}
        />
        <div
          className={`transition-[padding-left] duration-300 ease-in-out motion-reduce:transition-none print:pl-0 ${sidebarCollapsed ? "lg:pl-[72px]" : "lg:pl-64"}`}
        >
          <Topbar onMenuClick={() => setSidebarOpen(true)} />
          <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6 lg:px-8 print:max-w-none print:p-0">
            {children}
          </main>
        </div>
      </div>
    </AppProvider>
  );
}
