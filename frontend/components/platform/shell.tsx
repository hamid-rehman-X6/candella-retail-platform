import Link from "next/link";
import { ShieldCheck } from "lucide-react";
import type { PlatformAdmin } from "@/lib/platform-api";
import { PlatformNav } from "@/components/platform/nav";
import { initials } from "@/lib/user-name";

/**
 * The platform console frame.
 *
 * Deliberately *not* the amber customer app: slate surfaces and a rose accent,
 * plus a permanent "Platform" badge. When one person can be both a retailer and
 * Candella staff, the most valuable safety feature is never being unsure which
 * plane you're looking at.
 */
export function PlatformShell({
  admin,
  children,
}: {
  admin: PlatformAdmin;
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-slate-950 text-slate-100">
      <header className="sticky top-0 z-30 border-b border-slate-800 bg-slate-950/85 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-7xl items-center gap-4 px-4 sm:px-6 lg:px-8">
          <Link href="/platform" className="flex items-center gap-2.5">
            <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-rose-500/15 text-rose-400 ring-1 ring-rose-500/30">
              <ShieldCheck className="h-4.5 w-4.5" />
            </span>
            <span className="text-sm font-semibold tracking-tight">
              Candella
              <span className="ml-2 rounded-full bg-rose-500/15 px-2 py-0.5 text-[10px] font-medium tracking-wide text-rose-300 uppercase">
                Platform
              </span>
            </span>
          </Link>

          <PlatformNav />

          <div className="ml-auto flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <p className="text-xs font-medium text-slate-200">
                {admin.fullName}
              </p>
              <p className="text-[11px] text-slate-500 capitalize">
                {admin.role}
              </p>
            </div>
            <span
              title={admin.email}
              className="flex h-9 w-9 items-center justify-center rounded-full bg-rose-500/15 text-xs font-semibold text-rose-300 ring-1 ring-rose-500/30"
            >
              {initials(admin.fullName)}
            </span>
            <Link
              href="/dashboard"
              className="rounded-lg border border-slate-700 px-3 py-1.5 text-xs text-slate-300 transition-colors hover:border-slate-600 hover:text-white"
            >
              Exit
            </Link>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {children}
      </main>
    </div>
  );
}
