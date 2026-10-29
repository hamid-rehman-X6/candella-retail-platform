"use client";

import { useEffect, useState, type FormEvent } from "react";
import {
  Download,
  TrendingUp,
  ReceiptText,
  RotateCcw,
  Boxes,
} from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { Select } from "@/components/ui/select";
import { useStoreSelection } from "@/lib/use-store-selection";
import { formatMoney } from "@/lib/money";
import { businessToday } from "@/lib/datetime";
import {
  downloadDailyReportCSV,
  getDailyReport,
  type DailyReport,
} from "@/lib/reports-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "inline-flex cursor-pointer items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50";

function daysBefore(day: string, days: number) {
  const date = new Date(`${day}T12:00:00Z`);
  date.setUTCDate(date.getUTCDate() - days);
  return date.toISOString().slice(0, 10);
}

export default function ReportsPage() {
  const { current } = useApp();
  return <ReportsWorkspace key={current.id} />;
}

function ReportsWorkspace() {
  const { current } = useApp();
  const {
    stores,
    storeId,
    select,
    error: storeError,
  } = useStoreSelection(current.id);
  const canView = current.role === "owner" || current.role === "admin";
  const [from, setFrom] = useState(() =>
    daysBefore(businessToday(current.timezone), 6),
  );
  const [to, setTo] = useState(() => businessToday(current.timezone));
  const [applied, setApplied] = useState(() => ({
    from: daysBefore(businessToday(current.timezone), 6),
    to: businessToday(current.timezone),
  }));
  const [report, setReport] = useState<DailyReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!canView || !storeId) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      setLoading(true);
      setError("");
      void getDailyReport(current.id, storeId, applied.from, applied.to)
        .then(({ report }) => {
          if (!cancelled) setReport(report);
        })
        .catch((e) => {
          if (!cancelled) {
            setReport(null);
            setError(e instanceof Error ? e.message : "Could not load report.");
          }
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }, 0);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, storeId, applied, canView]);

  function apply(e: FormEvent) {
    e.preventDefault();
    const start = new Date(`${from}T00:00:00Z`);
    const end = new Date(`${to}T00:00:00Z`);
    if (
      !from ||
      !to ||
      end < start ||
      (end.getTime() - start.getTime()) / 86400000 > 365
    ) {
      setError("Choose a valid date range of up to 366 days.");
      return;
    }
    setApplied({ from, to });
  }
  async function download() {
    if (!report || exporting) return;
    setExporting(true);
    setError("");
    try {
      await downloadDailyReportCSV(
        current.id,
        storeId,
        applied.from,
        applied.to,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not export report.");
    } finally {
      setExporting(false);
    }
  }

  if (!canView)
    return (
      <div className="mx-auto max-w-5xl p-6 text-sm text-muted">
        Ask a workspace owner or admin for reports access.
      </div>
    );
  const money = (n: number) =>
    formatMoney(n, report?.currencyCode ?? current.currencyCode);
  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-12">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-3xl font-semibold">Reports</h1>
          <p className="mt-1 text-sm text-muted">
            Reconcile cash sales, refunds and stock movements by store and
            business day.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void download()}
          disabled={!report || loading || exporting}
          className={button}
        >
          <Download className="h-4 w-4" />
          {exporting ? "Exporting…" : "Export CSV"}
        </button>
      </div>
      <form
        onSubmit={apply}
        className="flex flex-wrap items-end gap-3 rounded-2xl border border-line bg-surface p-5"
      >
        <label className="min-w-44 flex-1 text-sm font-medium">
          Store
          <Select
            ariaLabel="Report store"
            className={`${field} mt-2`}
            value={storeId}
            onValueChange={(id) => {
              setReport(null);
              select(id);
            }}
            options={[
              ...(!storeId ? [{ value: "", label: "Select a store" }] : []),
              ...stores
                .filter((s) => s.accessible)
                .map((s) => ({ value: s.id, label: s.name })),
            ]}
          />
        </label>
        <label className="min-w-40 flex-1 text-sm font-medium">
          From
          <input
            className={`${field} mt-2`}
            type="date"
            value={from}
            max={to}
            onChange={(e) => setFrom(e.target.value)}
            required
          />
        </label>
        <label className="min-w-40 flex-1 text-sm font-medium">
          To
          <input
            className={`${field} mt-2`}
            type="date"
            value={to}
            min={from}
            onChange={(e) => setTo(e.target.value)}
            required
          />
        </label>
        <button type="submit" disabled={!storeId || loading} className={button}>
          Apply dates
        </button>
      </form>
      {(error || storeError) && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error || storeError}
        </p>
      )}
      {loading && <p className="text-sm text-muted">Calculating report…</p>}
      {report && (
        <>
          <p className="text-sm text-muted">
            {report.storeName} · {report.fromDate} to {report.toDate} ·{" "}
            {report.timezone}. Refunds are counted on the day they were paid,
            including returns of earlier sales. Net tax for the period:{" "}
            <span className="font-medium text-foreground">
              {money(report.totals.netTaxMinor)}
            </span>
            ; discounts given:{" "}
            <span className="font-medium text-foreground">
              {money(report.totals.discountMinor)}
            </span>
            .
          </p>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
            {[
              {
                label: "Sales",
                value: report.totals.salesCount,
                icon: ReceiptText,
              },
              {
                label: "Gross cash",
                value: money(report.totals.grossCashMinor),
                icon: TrendingUp,
              },
              {
                label: "Cash refunds",
                value: money(report.totals.refundsMinor),
                icon: RotateCcw,
              },
              {
                label: "Net sales",
                value: money(report.totals.netSalesMinor),
                icon: TrendingUp,
              },
            ].map(({ label, value, icon: Icon }) => (
              <div
                key={label}
                className="rounded-2xl border border-line bg-surface p-5"
              >
                <div className="flex items-center justify-between text-sm text-muted">
                  <span>{label}</span>
                  <Icon className="h-4 w-4 text-accent-ink" />
                </div>
                <p className="mt-3 text-2xl font-semibold">{value}</p>
              </div>
            ))}
          </div>
          <section className="overflow-hidden rounded-2xl border border-line bg-surface">
            <h2 className="border-b border-line p-5 text-lg font-semibold">
              Daily cash summary
            </h2>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[820px] text-left text-sm">
                <thead className="border-b border-line text-muted">
                  <tr>
                    {[
                      "Business day",
                      "Sales",
                      "Gross cash",
                      "Refunds",
                      "Net sales",
                      "Net tax",
                    ].map((h) => (
                      <th key={h} className="px-5 py-3 font-medium">
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {[...report.days].reverse().map((day) => (
                    <tr key={day.date}>
                      <td className="px-5 py-3 font-medium">{day.date}</td>
                      <td className="px-5 py-3">{day.salesCount}</td>
                      <td className="px-5 py-3">{money(day.grossCashMinor)}</td>
                      <td className="px-5 py-3">
                        {money(day.refundsMinor)}
                        {day.refundsCount > 0 && (
                          <span className="block text-xs text-muted">
                            {day.refundsCount} return
                            {day.refundsCount === 1 ? "" : "s"}
                          </span>
                        )}
                      </td>
                      <td className="px-5 py-3 font-semibold">
                        {money(day.netSalesMinor)}
                      </td>
                      <td className="px-5 py-3">
                        {money(day.netTaxMinor)}
                        {day.refundedTaxMinor > 0 && (
                          <span className="block text-xs text-muted">
                            {money(day.taxMinor)} charged,{" "}
                            {money(day.refundedTaxMinor)} refunded
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          <div className="grid gap-6 lg:grid-cols-2">
            <section className="rounded-2xl border border-line bg-surface p-5">
              <h2 className="text-lg font-semibold">Top products</h2>
              <p className="mt-1 text-sm text-muted">
                Ranked by gross cash sales in this period.
              </p>
              {report.topProducts.length === 0 ? (
                <p className="mt-5 text-sm text-muted">
                  No products sold in this period.
                </p>
              ) : (
                <div className="mt-4 space-y-3">
                  {report.topProducts.map((product, i) => (
                    <div
                      key={product.productId}
                      className="flex items-center justify-between gap-3 border-b border-line pb-3 last:border-0"
                    >
                      <div className="text-sm">
                        <span className="mr-2 text-muted">{i + 1}.</span>
                        <span className="font-medium">
                          {product.productName}
                        </span>
                        <span className="block text-xs text-muted">
                          {product.unitsSold} units sold
                        </span>
                      </div>
                      <span className="text-sm font-semibold">
                        {money(product.grossSalesMinor)}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </section>
            <section className="rounded-2xl border border-line bg-surface p-5">
              <div className="flex items-center gap-2">
                <Boxes className="h-5 w-5 text-accent-ink" />
                <h2 className="text-lg font-semibold">Stock movements</h2>
              </div>
              <p className="mt-1 text-sm text-muted">
                Units moved during this period at {report.storeName}.
              </p>
              {report.stockMovements.length === 0 ? (
                <p className="mt-5 text-sm text-muted">
                  No stock movements in this period.
                </p>
              ) : (
                <div className="mt-4 space-y-3">
                  {report.stockMovements.map((movement) => (
                    <div
                      key={movement.kind}
                      className="flex items-center justify-between gap-3 border-b border-line pb-3 text-sm last:border-0"
                    >
                      <div>
                        <span className="font-medium capitalize">
                          {movement.kind}
                        </span>
                        <span className="block text-xs text-muted">
                          {movement.movements} movement
                          {movement.movements === 1 ? "" : "s"}
                        </span>
                      </div>
                      <div className="text-right">
                        <span className="text-success-ink">
                          +{movement.unitsIn}
                        </span>
                        <span className="mx-2 text-muted">/</span>
                        <span className="text-warning-ink">
                          −{movement.unitsOut}
                        </span>
                        <span className="block text-xs text-muted">
                          Net {movement.netUnits >= 0 ? "+" : ""}
                          {movement.netUnits}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </section>
          </div>
        </>
      )}
    </div>
  );
}
