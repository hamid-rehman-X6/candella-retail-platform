"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Package,
  Store,
  Boxes,
  Truck,
  Users,
  BarChart3,
  Settings,
  PanelLeftClose,
  PanelLeftOpen,
  X,
  type LucideIcon,
} from "lucide-react";
import { LogoMark } from "@/components/ui/logo";
import { useApp, workspacePath } from "@/components/app/app-context";
import { cn } from "@/lib/cn";

// Paths are relative to the current workspace; the sidebar prefixes them.
const mainNav: { label: string; path: string; icon: LucideIcon }[] = [
  { label: "Overview", path: "", icon: LayoutDashboard },
  { label: "Products", path: "/products", icon: Package },
  { label: "Inventory", path: "/inventory", icon: Boxes },
  { label: "Suppliers", path: "/suppliers", icon: Truck },
  { label: "Sales", path: "/sales", icon: Store },
];

// Business modules ship later; shown now so the shell reads as a real app.
const moduleNav: { label: string; icon: LucideIcon }[] = [
  { label: "Customers", icon: Users },
  { label: "Reports", icon: BarChart3 },
];

function NavLabel({
  children,
  collapsed,
}: {
  children: React.ReactNode;
  collapsed: boolean;
}) {
  return (
    <span
      className={cn(
        "max-w-36 overflow-hidden whitespace-nowrap transition-[max-width,opacity] duration-300 motion-reduce:transition-none",
        collapsed && "lg:max-w-0 lg:opacity-0",
      )}
    >
      {children}
    </span>
  );
}

function SoonItem({
  label,
  icon: Icon,
  collapsed,
}: {
  label: string;
  icon: LucideIcon;
  collapsed: boolean;
}) {
  return (
    <span
      title={collapsed ? `${label} (Soon)` : undefined}
      className={cn(
        "flex cursor-default items-center justify-between rounded-xl px-3 py-2.5 text-sm text-subtle transition-all duration-300 motion-reduce:transition-none",
        collapsed && "lg:justify-center lg:gap-0",
      )}
    >
      <span
        className={cn(
          "flex min-w-0 items-center gap-3",
          collapsed && "lg:gap-0",
        )}
      >
        <Icon className="h-4.5 w-4.5 shrink-0" />
        <NavLabel collapsed={collapsed}>{label}</NavLabel>
      </span>
      <span
        className={cn(
          "max-w-16 overflow-hidden rounded-full bg-foreground/5 px-2 py-0.5 text-[10px] tracking-wide text-subtle transition-[max-width,opacity,padding] duration-300 motion-reduce:transition-none",
          collapsed && "lg:max-w-0 lg:px-0 lg:opacity-0",
        )}
      >
        Soon
      </span>
    </span>
  );
}

export function Sidebar({
  open,
  onClose,
  collapsed,
  onToggleCollapsed,
}: {
  open: boolean;
  onClose: () => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
}) {
  const pathname = usePathname();
  const { current } = useApp();
  const home = workspacePath(current.slug);
  const canManageStaff = current.role === "owner" || current.role === "admin";

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
        id="app-sidebar"
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-line bg-canvas-raised transition-[transform,width] duration-300 ease-in-out motion-reduce:transition-none lg:translate-x-0 print:hidden",
          open ? "translate-x-0" : "-translate-x-full",
          collapsed && "lg:w-[72px]",
        )}
      >
        <div className="flex h-16 shrink-0 items-center justify-between px-5 lg:px-4">
          <Link
            href={home}
            className={cn(
              "flex min-w-0 items-center gap-2.5 overflow-hidden transition-[width,opacity,visibility] duration-300 motion-reduce:transition-none",
              collapsed
                ? "lg:pointer-events-none lg:invisible lg:w-0 lg:opacity-0"
                : "lg:w-40",
            )}
            onClick={onClose}
          >
            <LogoMark className="h-8 w-8 shrink-0" />
            <span className="text-lg font-semibold tracking-tight whitespace-nowrap text-foreground">
              Candella
            </span>
          </Link>
          <button
            type="button"
            onClick={onToggleCollapsed}
            aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            aria-controls="app-sidebar"
            aria-expanded={!collapsed}
            title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            className="hidden h-9 w-9 shrink-0 cursor-pointer items-center justify-center rounded-lg text-muted transition-colors hover:bg-foreground/5 hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent lg:flex"
          >
            {collapsed ? (
              <PanelLeftOpen className="h-5 w-5" />
            ) : (
              <PanelLeftClose className="h-5 w-5" />
            )}
          </button>
          <button
            onClick={onClose}
            aria-label="Close menu"
            className="flex h-9 w-9 items-center justify-center rounded-lg text-muted transition-colors hover:bg-foreground/5 hover:text-foreground lg:hidden"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {mainNav.map((item) => {
            const href = workspacePath(current.slug, item.path);
            const active = pathname === href;
            const Icon = item.icon;
            return (
              <Link
                key={item.label}
                href={href}
                onClick={onClose}
                aria-label={collapsed ? item.label : undefined}
                title={collapsed ? item.label : undefined}
                className={cn(
                  "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300 motion-reduce:transition-none",
                  collapsed && "lg:justify-center lg:gap-0",
                  active
                    ? "bg-accent/10 text-accent-ink ring-1 ring-accent/20"
                    : "text-muted hover:bg-foreground/5 hover:text-foreground",
                )}
              >
                <Icon className="h-4.5 w-4.5 shrink-0" />
                <NavLabel collapsed={collapsed}>{item.label}</NavLabel>
              </Link>
            );
          })}

          <p
            className={cn(
              "max-h-16 overflow-hidden px-3 pt-5 pb-2 text-xs font-medium tracking-wider text-subtle uppercase transition-[max-height,padding,opacity] duration-300 motion-reduce:transition-none",
              collapsed && "lg:max-h-0 lg:pt-0 lg:pb-0 lg:opacity-0",
            )}
          >
            Modules
          </p>
          {moduleNav.map((item) => (
            <SoonItem
              key={item.label}
              label={item.label}
              icon={item.icon}
              collapsed={collapsed}
            />
          ))}
          {canManageStaff && (
            <Link
              href={workspacePath(current.slug, "/stores")}
              onClick={onClose}
              aria-label={collapsed ? "Stores" : undefined}
              title={collapsed ? "Stores" : undefined}
              className={cn(
                "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300 motion-reduce:transition-none",
                collapsed && "lg:justify-center lg:gap-0",
                pathname === workspacePath(current.slug, "/stores")
                  ? "bg-accent/10 text-accent-ink ring-1 ring-accent/20"
                  : "text-muted hover:bg-foreground/5 hover:text-foreground",
              )}
            >
              <Store className="h-4.5 w-4.5 shrink-0" />
              <NavLabel collapsed={collapsed}>Stores</NavLabel>
            </Link>
          )}
          {canManageStaff && (
            <Link
              href={workspacePath(current.slug, "/staff")}
              onClick={onClose}
              aria-label={collapsed ? "Staff" : undefined}
              title={collapsed ? "Staff" : undefined}
              className={cn(
                "flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-all duration-300 motion-reduce:transition-none",
                collapsed && "lg:justify-center lg:gap-0",
                pathname === workspacePath(current.slug, "/staff")
                  ? "bg-accent/10 text-accent-ink ring-1 ring-accent/20"
                  : "text-muted hover:bg-foreground/5 hover:text-foreground",
              )}
            >
              <Users className="h-4.5 w-4.5 shrink-0" />
              <NavLabel collapsed={collapsed}>Staff</NavLabel>
            </Link>
          )}
        </nav>

        <div className="border-t border-line p-3">
          <SoonItem label="Settings" icon={Settings} collapsed={collapsed} />
        </div>
      </aside>
    </>
  );
}
