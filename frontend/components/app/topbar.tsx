"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Menu,
  ChevronsUpDown,
  Check,
  Plus,
  LogOut,
  Loader2,
} from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { logout } from "@/lib/auth-api";
import { clearSession } from "@/lib/session-store";
import { toast } from "@/lib/toast";
import { industryLabel } from "@/lib/industries";
import { initials } from "@/lib/user-name";
import { cn } from "@/lib/cn";

export function Topbar({ onMenuClick }: { onMenuClick: () => void }) {
  const router = useRouter();
  const { user, workspaces, current } = useApp();
  const [wsOpen, setWsOpen] = useState(false);
  const [userOpen, setUserOpen] = useState(false);
  const [signingOut, setSigningOut] = useState(false);

  async function handleLogout() {
    setSigningOut(true);
    try {
      await logout();
    } catch {
      /* even if the call fails, send them to login */
    }
    clearSession(); // so the public pages stop showing the Dashboard shortcut
    toast.success("Signed out", { description: "See you next time." });
    router.push("/login");
    router.refresh();
  }

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center gap-3 border-b border-line bg-canvas/80 px-4 backdrop-blur-xl sm:px-6 print:hidden">
      <button
        onClick={onMenuClick}
        aria-label="Open menu"
        className="flex h-9 w-9 items-center justify-center rounded-lg text-muted transition-colors hover:bg-white/5 hover:text-foreground lg:hidden"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Workspace switcher */}
      <div className="relative">
        <button
          onClick={() => setWsOpen((v) => !v)}
          className="flex items-center gap-2.5 rounded-xl border border-line bg-white/3 px-3 py-2 text-left transition-colors hover:border-line-strong"
        >
          <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-linear-to-br from-accent-bright to-accent-deep text-xs font-bold text-black">
            {current.name.charAt(0).toUpperCase()}
          </span>
          <span className="hidden sm:block">
            <span className="block max-w-48 truncate text-sm font-medium text-foreground">
              {current.name}
            </span>
            <span className="block text-[11px] text-subtle">
              {industryLabel(current.industryType)}
            </span>
          </span>
          <ChevronsUpDown className="h-4 w-4 text-subtle" />
        </button>

        {wsOpen && (
          <>
            <div
              className="fixed inset-0 z-40"
              onClick={() => setWsOpen(false)}
            />
            <div className="glass absolute top-full left-0 z-50 mt-2 w-64 overflow-hidden rounded-2xl p-1.5 shadow-xl">
              <p className="px-3 py-2 text-xs tracking-wide text-subtle uppercase">
                Workspaces
              </p>
              {/* Switching is navigation, not state — the URL is the workspace. */}
              {workspaces.map((ws) => (
                <Link
                  key={ws.id}
                  href={workspacePath(ws.slug)}
                  onClick={() => setWsOpen(false)}
                  className="flex w-full items-center justify-between gap-2 rounded-xl px-3 py-2 text-left text-sm transition-colors hover:bg-white/5"
                >
                  <span className="min-w-0">
                    <span className="block truncate font-medium text-foreground">
                      {ws.name}
                    </span>
                    <span className="block text-[11px] text-subtle">
                      {industryLabel(ws.industryType)} · {ws.role}
                    </span>
                  </span>
                  {ws.id === current.id && (
                    <Check className="h-4 w-4 shrink-0 text-accent" />
                  )}
                </Link>
              ))}
              <div className="my-1 h-px bg-line" />
              <Link
                href={`/onboarding?from=${encodeURIComponent(current.slug)}`}
                onClick={() => setWsOpen(false)}
                className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-muted transition-colors hover:bg-white/5 hover:text-foreground"
              >
                <Plus className="h-4 w-4" />
                New workspace
              </Link>
            </div>
          </>
        )}
      </div>

      <div className="ml-auto" />

      {/* User menu */}
      <div className="relative">
        <button
          onClick={() => setUserOpen((v) => !v)}
          aria-label="Account menu"
          className="flex h-9 w-9 items-center justify-center rounded-full bg-accent/15 text-sm font-semibold text-accent ring-1 ring-accent/25 transition-transform hover:scale-105"
        >
          {initials(user.fullName)}
        </button>

        {userOpen && (
          <>
            <div
              className="fixed inset-0 z-40"
              onClick={() => setUserOpen(false)}
            />
            <div className="glass absolute top-full right-0 z-50 mt-2 w-60 overflow-hidden rounded-2xl p-1.5 shadow-xl">
              <div className="px-3 py-2.5">
                <p className="truncate text-sm font-medium text-foreground">
                  {user.fullName}
                </p>
                <p className="truncate text-xs text-subtle">{user.email}</p>
              </div>
              <div className="my-1 h-px bg-line" />
              <button
                onClick={handleLogout}
                disabled={signingOut}
                className={cn(
                  "flex w-full items-center gap-2 rounded-xl px-3 py-2 text-left text-sm text-muted transition-colors hover:bg-white/5 hover:text-foreground",
                  signingOut && "opacity-60",
                )}
              >
                {signingOut ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <LogOut className="h-4 w-4" />
                )}
                Sign out
              </button>
            </div>
          </>
        )}
      </div>
    </header>
  );
}
