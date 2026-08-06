"use client";

import { Receipt, Store, Package, Users, Check, Sparkles } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { industryLabel } from "@/lib/industries";

const kpis = [
  { label: "Revenue", value: "—", icon: Receipt },
  { label: "Orders", value: "0", icon: Store },
  { label: "Products", value: "0", icon: Package },
  { label: "Customers", value: "0", icon: Users },
];

const nextSteps = [
  { label: "Add your products", done: false },
  { label: "Set up a register (POS)", done: false },
  { label: "Invite your team", done: false },
];

export default function DashboardPage() {
  const { user, currentWorkspace } = useApp();
  const firstName = user.fullName.split(" ")[0] || user.fullName;

  return (
    <div className="flex flex-col gap-8">
      {/* Greeting */}
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          Welcome, {firstName} 👋
        </h1>
        <p className="mt-1 text-muted">
          Here&apos;s what&apos;s happening at{" "}
          <span className="text-foreground/90">{currentWorkspace.name}</span> ·{" "}
          {industryLabel(currentWorkspace.industryType)}
        </p>
      </div>

      {/* KPI tiles (empty states until the business modules ship) */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {kpis.map((kpi) => {
          const Icon = kpi.icon;
          return (
            <div
              key={kpi.label}
              className="rounded-2xl border border-line bg-white/[0.02] p-5"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted">{kpi.label}</span>
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent/10 text-accent">
                  <Icon className="h-4 w-4" />
                </span>
              </div>
              <p className="mt-3 text-3xl font-semibold tracking-tight">
                {kpi.value}
              </p>
              <p className="mt-1 text-xs text-subtle">No data yet</p>
            </div>
          );
        })}
      </div>

      {/* Getting started */}
      <div className="card-surface ring-gradient relative overflow-hidden rounded-3xl p-6 sm:p-8">
        <div className="absolute -top-16 -right-16 h-40 w-40 rounded-full bg-[radial-gradient(circle,rgba(245,158,11,0.14),transparent_70%)] blur-2xl" />
        <div className="relative flex items-start gap-4">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-accent/10 text-accent ring-1 ring-accent/20">
            <Sparkles className="h-5 w-5" />
          </span>
          <div className="flex-1">
            <h2 className="text-lg font-semibold tracking-tight">
              Your workspace is ready
            </h2>
            <p className="mt-1 text-sm text-muted">
              {currentWorkspace.name} is set up. The retail modules (POS,
              inventory, customers and reports) are on the way — here&apos;s
              what&apos;s next once they land.
            </p>

            <ul className="mt-5 flex flex-col gap-2.5">
              <li className="flex items-center gap-3 text-sm">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400">
                  <Check className="h-3 w-3" />
                </span>
                <span className="text-foreground/85">
                  Create your workspace
                </span>
              </li>
              {nextSteps.map((step) => (
                <li
                  key={step.label}
                  className="flex items-center gap-3 text-sm"
                >
                  <span className="h-5 w-5 rounded-full border border-line" />
                  <span className="text-muted">{step.label}</span>
                  <span className="ml-auto rounded-full bg-white/5 px-2 py-0.5 text-[10px] tracking-wide text-subtle">
                    Soon
                  </span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
