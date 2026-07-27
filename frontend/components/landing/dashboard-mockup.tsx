import {
  LayoutDashboard,
  Store,
  Boxes,
  Users,
  BarChart3,
  Settings,
  TrendingUp,
  ArrowUpRight,
} from "lucide-react";
import { LogoMark } from "@/components/ui/logo";

const bars = [42, 58, 47, 72, 63, 88, 76];
const days = ["M", "T", "W", "T", "F", "S", "S"];

const feed = [
  { store: "Downtown Pharmacy", amount: "$248.50", tag: "POS" },
  { store: "Mall Kiosk · Beauty", amount: "$92.00", tag: "Online" },
  { store: "Warehouse Outlet", amount: "$1,204.00", tag: "Wholesale" },
];

/** Stylised product screenshot used in the hero. Pure CSS/SVG, no images. */
export function DashboardMockup() {
  return (
    <div className="card-surface ring-gradient relative overflow-hidden rounded-2xl">
      {/* window chrome */}
      <div className="flex items-center gap-2 border-b border-line px-4 py-3">
        <span className="h-3 w-3 rounded-full bg-red-400/70" />
        <span className="h-3 w-3 rounded-full bg-amber-400/70" />
        <span className="h-3 w-3 rounded-full bg-emerald-400/70" />
        <div className="mx-auto flex items-center gap-2 rounded-full border border-line bg-white/5 px-3 py-1 text-[11px] text-subtle">
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
          app.candella.com/dashboard
        </div>
      </div>

      <div className="flex">
        {/* sidebar */}
        <aside className="hidden w-14 shrink-0 flex-col items-center gap-4 border-r border-line py-5 sm:flex">
          <LogoMark className="h-8 w-8" />
          {[LayoutDashboard, Store, Boxes, Users, BarChart3, Settings].map(
            (Icon, i) => (
              <span
                key={i}
                className={`flex h-9 w-9 items-center justify-center rounded-xl ${
                  i === 0
                    ? "bg-accent/15 text-accent"
                    : "text-subtle hover:text-muted"
                }`}
              >
                <Icon className="h-4 w-4" />
              </span>
            ),
          )}
        </aside>

        {/* main */}
        <div className="min-w-0 flex-1 p-4 sm:p-6">
          <div className="mb-5 flex items-center justify-between">
            <div>
              <p className="text-xs text-subtle">Good morning, Ayesha</p>
              <h3 className="text-base font-semibold">Business overview</h3>
            </div>
            <div className="rounded-full border border-line bg-white/5 px-3 py-1 text-[11px] text-muted">
              Last 7 days
            </div>
          </div>

          {/* KPI tiles */}
          <div className="grid grid-cols-3 gap-3">
            {[
              { label: "Revenue", value: "$48.2k", delta: "+12.4%" },
              { label: "Orders", value: "1,284", delta: "+6.1%" },
              { label: "Stock health", value: "98%", delta: "+2.0%" },
            ].map((kpi) => (
              <div
                key={kpi.label}
                className="rounded-xl border border-line bg-canvas-raised/60 p-3"
              >
                <p className="text-[11px] text-subtle">{kpi.label}</p>
                <p className="mt-1 text-lg font-semibold tracking-tight">
                  {kpi.value}
                </p>
                <p className="mt-0.5 inline-flex items-center gap-0.5 text-[10px] text-emerald-400">
                  <TrendingUp className="h-3 w-3" />
                  {kpi.delta}
                </p>
              </div>
            ))}
          </div>

          {/* chart */}
          <div className="mt-4 rounded-xl border border-line bg-canvas-raised/60 p-4">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-xs font-medium text-muted">Weekly sales</p>
              <span className="inline-flex items-center gap-1 text-[10px] text-accent">
                <ArrowUpRight className="h-3 w-3" /> trending up
              </span>
            </div>
            <div className="flex h-24 items-end gap-2">
              {bars.map((h, i) => (
                <div
                  key={i}
                  className="flex flex-1 flex-col items-center gap-1.5"
                >
                  <div className="flex w-full flex-1 items-end">
                    <div
                      className="w-full rounded-md bg-linear-to-t from-accent/30 to-accent-bright"
                      style={{ height: `${h}%` }}
                    />
                  </div>
                  <span className="text-[9px] text-subtle">{days[i]}</span>
                </div>
              ))}
            </div>
          </div>

          {/* live feed */}
          <div className="mt-4 rounded-xl border border-line bg-canvas-raised/60 p-4">
            <p className="mb-3 text-xs font-medium text-muted">Live sales</p>
            <ul className="space-y-2.5">
              {feed.map((row) => (
                <li
                  key={row.store}
                  className="flex items-center justify-between"
                >
                  <div className="flex items-center gap-2">
                    <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                    <span className="text-xs text-foreground/90">
                      {row.store}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="rounded-full border border-line px-1.5 py-0.5 text-[9px] text-subtle">
                      {row.tag}
                    </span>
                    <span className="text-xs font-medium">{row.amount}</span>
                  </div>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
