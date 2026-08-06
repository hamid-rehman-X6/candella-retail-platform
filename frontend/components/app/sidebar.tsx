"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Store,
  Boxes,
  Users,
  BarChart3,
  Settings,
  X,
  type LucideIcon,
} from "lucide-react";
import { LogoMark } from "@/components/ui/logo";
import { cn } from "@/lib/cn";

const mainNav: { label: string; href: string; icon: LucideIcon }[] = [
  { label: "Overview", href: "/dashboard", icon: LayoutDashboard },
];

// Business modules ship later; shown now so the shell reads as a real app.
const moduleNav: { label: string; icon: LucideIcon }[] = [
  { label: "Sales", icon: Store },
  { label: "Inventory", icon: Boxes },
  { label: "Customers", icon: Users },
  { label: "Reports", icon: BarChart3 },
];

function SoonItem({ label, icon: Icon }: { label: string; icon: LucideIcon }) {
  return (
    <span className="flex cursor-default items-center justify-between rounded-xl px-3 py-2.5 text-sm text-subtle">
      <span className="flex items-center gap-3">
        <Icon className="h-4.5 w-4.5" />
        {label}
      </span>
      <span className="rounded-full bg-white/5 px-2 py-0.5 text-[10px] tracking-wide text-subtle">
        Soon
      </span>
    </span>
  );
}

export function Sidebar({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const pathname = usePathname();

  return (
    <>
      {/* Mobile overlay */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm lg:hidden"
          onClick={onClose}
          aria-hidden
        />
      )}

      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-line bg-canvas-raised transition-transform duration-300 lg:translate-x-0",
          open ? "translate-x-0" : "-translate-x-full",
        )}
      >
        <div className="flex h-16 shrink-0 items-center justify-between px-5">
          <Link
            href="/dashboard"
            className="flex items-center gap-2.5"
            onClick={onClose}
          >
            <LogoMark className="h-8 w-8" />
            <span className="text-lg font-semibold tracking-tight text-foreground">
              Candella
            </span>
          </Link>
          <button
            onClick={onClose}
            aria-label="Close menu"
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted transition-colors hover:bg-white/5 hover:text-foreground lg:hidden"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {mainNav.map((item) => {
            const active = pathname === item.href;
            const Icon = item.icon;
            return (
              <Link
                key={item.label}
                href={item.href}
                onClick={onClose}
                className={cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors",
                  active
                    ? "bg-accent/10 text-accent ring-1 ring-accent/20"
                    : "text-muted hover:bg-white/5 hover:text-foreground",
                )}
              >
                <Icon className="h-4.5 w-4.5" />
                {item.label}
              </Link>
            );
          })}

          <p className="px-3 pt-5 pb-2 text-xs font-medium tracking-wider text-subtle uppercase">
            Modules
          </p>
          {moduleNav.map((item) => (
            <SoonItem key={item.label} label={item.label} icon={item.icon} />
          ))}
        </nav>

        <div className="border-t border-line p-3">
          <SoonItem label="Settings" icon={Settings} />
        </div>
      </aside>
    </>
  );
}
