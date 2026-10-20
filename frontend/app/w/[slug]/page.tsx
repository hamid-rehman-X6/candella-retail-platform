"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Receipt, Store, Package, Boxes, Check, Sparkles } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { industryLabel } from "@/lib/industries";
import { getCatalogSummary, type CatalogSummary } from "@/lib/catalog-api";
import { useStoreSelection } from "@/lib/use-store-selection";
import { businessToday } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import { getDailyReport, type DailyReport } from "@/lib/reports-api";

export default function DashboardPage() {
  const { user, current } = useApp();
  const { storeId, selectedStore } = useStoreSelection(current.id);
  const firstName = user.fullName.split(" ")[0] || user.fullName;
  const [summary, setSummary] = useState<CatalogSummary | null>(null);
  const [dailyReport, setDailyReport] = useState<DailyReport | null>(null);
  const canViewReports = current.role === "owner" || current.role === "admin";
  const today = businessToday(current.timezone);

  useEffect(() => {
    if (!storeId) return;
    let active = true;
    getCatalogSummary(current.id, storeId)
      .then(({ summary: result }) => {
        if (active) setSummary(result);
      })
      .catch(() => {
        if (active) setSummary(null);
      });
    return () => {
      active = false;
    };
  }, [current.id, storeId]);

  useEffect(() => {
    if (!storeId || !canViewReports) return;
    let active = true;
    getDailyReport(current.id, storeId, today, today)
      .then(({ report }) => {
        if (active) setDailyReport(report);
      })
      .catch(() => {
        if (active) setDailyReport(null);
      });
    return () => {
      active = false;
    };
  }, [current.id, storeId, today, canViewReports]);

  const todayTotals =
    dailyReport?.storeId === storeId && dailyReport.fromDate === today
      ? dailyReport.totals
      : null;

  const kpis = [
    {
      label: "Net sales",
      value: todayTotals
        ? formatMoney(todayTotals.netSalesMinor, current.currencyCode)
        : "—",
      icon: Receipt,
      note: canViewReports ? "Today · after cash refunds" : "Manager access",
    },
    {
      label: "Sales",
      value: todayTotals?.salesCount ?? "—",
      icon: Store,
      note: canViewReports ? "Completed today" : "Manager access",
    },
    {
      label: "Products",
      value: summary?.productCount ?? "—",
      icon: Package,
      note: "Active products",
    },
    {
      label: "Stock units",
      value: summary?.unitsOnHand ?? "—",
      icon: Boxes,
      note: selectedStore ? `At ${selectedStore.name}` : "Select a store",
    },
  ];

  return (
    <div className="flex flex-col gap-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          Welcome, {firstName}
        </h1>
        <p className="mt-1 text-muted">
          Here&apos;s what&apos;s happening at{" "}
          <span className="text-foreground/90">{current.name}</span> ·{" "}
          {industryLabel(current.industryType)}
        </p>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {kpis.map((kpi) => {
          const Icon = kpi.icon;
          return (
            <div
              key={kpi.label}
              className="rounded-2xl border border-line bg-foreground/[0.02] p-5"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm text-muted">{kpi.label}</span>
                <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent/10 text-accent-ink">
                  <Icon className="h-4 w-4" />
                </span>
              </div>
              <p className="mt-3 text-3xl font-semibold tracking-tight">
                {kpi.value}
              </p>
              <p className="mt-1 text-xs text-subtle">{kpi.note}</p>
            </div>
          );
        })}
      </div>
      <div className="card-surface ring-gradient relative overflow-hidden rounded-3xl p-6 sm:p-8">
        <div className="flex items-start gap-4">
          <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl bg-accent/10 text-accent-ink ring-1 ring-accent/20">
            <Sparkles className="h-5 w-5" />
          </span>
          <div>
            <h2 className="text-lg font-semibold tracking-tight">
              Set up your store
            </h2>
            <p className="mt-1 text-sm text-muted">
              Add products and starting stock, then use Sales for checkout.
              Managers can review each day in Reports.
            </p>
            <ul className="mt-5 flex flex-col gap-3 text-sm">
              <li className="flex items-center gap-3">
                <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-500/20 text-success-ink">
                  <Check className="h-3 w-3" />
                </span>
                Create your workspace
              </li>
              <li className="flex items-center gap-3">
                <span
                  className={`h-5 w-5 rounded-full border ${summary?.productCount ? "border-emerald-500 bg-emerald-500/20" : "border-line"}`}
                />
                <Link
                  className="hover:text-accent-ink"
                  href={workspacePath(current.slug, "/products")}
                >
                  Add your products →
                </Link>
              </li>
              <li className="flex items-center gap-3">
                <span
                  className={`h-5 w-5 rounded-full border ${summary?.unitsOnHand ? "border-emerald-500 bg-emerald-500/20" : "border-line"}`}
                />
                <Link
                  className="hover:text-accent-ink"
                  href={workspacePath(current.slug, "/inventory")}
                >
                  Receive starting stock →
                </Link>
              </li>
              {canViewReports && (
                <li className="flex items-center gap-3">
                  <span className="h-5 w-5 rounded-full border border-line" />
                  <Link
                    className="hover:text-accent-ink"
                    href={workspacePath(current.slug, "/reports")}
                  >
                    Review daily reports →
                  </Link>
                </li>
              )}
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}
