"use client";

import Link from "next/link";
import { ArrowRight, LayoutDashboard } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useSession } from "@/lib/session-store";
import { initials } from "@/lib/user-name";
import { cn } from "@/lib/cn";

/**
 * The auth-aware region of the public navbar. Signed-out visitors keep the
 * Sign in / Get started pair; signed-in visitors instead see themselves and a
 * one-click way back into the app — so returning to the marketing site is never
 * a dead end.
 */

const avatar =
  "flex shrink-0 items-center justify-center rounded-full bg-accent/15 font-semibold text-accent ring-1 ring-accent/25";

/** Desktop actions (right side of the nav pill). */
export function NavAuth() {
  const session = useSession();

  if (session.status === "loading") {
    // Neutral placeholder at roughly the real width, so the pill never jumps.
    return (
      <div className="hidden items-center gap-2 lg:flex" aria-hidden>
        <div className="h-9 w-20 animate-pulse rounded-full bg-white/5" />
        <div className="h-9 w-32 animate-pulse rounded-full bg-white/[0.07]" />
      </div>
    );
  }

  if (session.status === "guest") {
    return (
      <div className="hidden items-center gap-2 lg:flex">
        <Button href="/login" variant="ghost" size="sm">
          Sign in
        </Button>
        <Button href="/signup" size="sm">
          Get started
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
      </div>
    );
  }

  const { user } = session;
  return (
    <div className="hidden items-center gap-2.5 lg:flex">
      <span
        title={`Signed in as ${user.fullName} (${user.email})`}
        className={cn(avatar, "h-9 w-9 text-xs")}
      >
        {initials(user.fullName)}
      </span>
      <Button href="/dashboard" size="sm">
        <LayoutDashboard className="h-4 w-4" />
        Dashboard
      </Button>
    </div>
  );
}

/** Compact avatar shown next to the hamburger on small screens. */
export function NavAuthAvatar() {
  const session = useSession();
  if (session.status !== "authenticated") return null;

  return (
    <Link
      href="/dashboard"
      aria-label="Go to your dashboard"
      className={cn(
        avatar,
        "h-9 w-9 text-xs transition-transform hover:scale-105 lg:hidden",
      )}
    >
      {initials(session.user.fullName)}
    </Link>
  );
}

/** Auth block inside the open mobile menu. */
export function NavAuthMobile({ onNavigate }: { onNavigate: () => void }) {
  const session = useSession();

  if (session.status === "loading") {
    return (
      <div
        className="mt-6 h-13 animate-pulse rounded-full bg-white/5"
        aria-hidden
      />
    );
  }

  if (session.status === "guest") {
    return (
      <div className="mt-6 flex flex-col gap-3">
        <Button
          href="/login"
          variant="secondary"
          size="lg"
          onClick={onNavigate}
        >
          Sign in
        </Button>
        <Button href="/signup" size="lg" onClick={onNavigate}>
          Get started
          <ArrowRight className="h-4 w-4" />
        </Button>
      </div>
    );
  }

  const { user } = session;
  return (
    <div className="mt-6 flex flex-col gap-3">
      <div className="flex items-center gap-3 rounded-2xl border border-line bg-white/[0.03] px-4 py-3">
        <span className={cn(avatar, "h-10 w-10 text-sm")}>
          {initials(user.fullName)}
        </span>
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-foreground">
            {user.fullName}
          </p>
          <p className="truncate text-xs text-subtle">{user.email}</p>
        </div>
      </div>
      <Button href="/dashboard" size="lg" onClick={onNavigate}>
        <LayoutDashboard className="h-4 w-4" />
        Go to dashboard
      </Button>
    </div>
  );
}
