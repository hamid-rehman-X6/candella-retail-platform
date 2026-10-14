"use client";

import Link from "next/link";
import { useEffect, useState, type FormEvent } from "react";
import { ArrowLeft, Search } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { formatMoney, parseMoney } from "@/lib/catalog-api";
import {
  listSales,
  type SaleSummary,
  type SalesFilters,
} from "@/lib/sales-api";

const pageSize = 20;
const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

type SearchForm = {
  fromDate: string;
  toDate: string;
  orderNumber: string;
  cashier: string;
  minAmount: string;
  maxAmount: string;
};

const emptyForm: SearchForm = {
  fromDate: "",
  toDate: "",
  orderNumber: "",
  cashier: "",
  minAmount: "",
  maxAmount: "",
};

export default function SalesHistoryPage() {
  const { current } = useApp();
  const [form, setForm] = useState<SearchForm>(emptyForm);
  const [filters, setFilters] = useState<SalesFilters>({});
  const [offset, setOffset] = useState(0);
  const [sales, setSales] = useState<SaleSummary[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    void listSales(current.id, { ...filters, offset, limit: pageSize })
      .then((result) => {
        if (cancelled) return;
        setSales(result.sales);
        setHasMore(result.hasMore);
        setError("");
      })
      .catch((cause) => {
        if (cancelled) return;
        setSales([]);
        setHasMore(false);
        setError(
          cause instanceof Error ? cause.message : "Could not load sales.",
        );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, filters, offset]);

  function search(event: FormEvent) {
    event.preventDefault();
    const minAmountMinor = form.minAmount
      ? parseMoney(form.minAmount, current.currencyCode)
      : undefined;
    const maxAmountMinor = form.maxAmount
      ? parseMoney(form.maxAmount, current.currencyCode)
      : undefined;
    if (
      (form.minAmount && minAmountMinor === null) ||
      (form.maxAmount && maxAmountMinor === null) ||
      (typeof minAmountMinor === "number" &&
        typeof maxAmountMinor === "number" &&
        minAmountMinor > maxAmountMinor) ||
      (form.fromDate && form.toDate && form.fromDate > form.toDate)
    ) {
      setError("Check the dates and amount range, then search again.");
      return;
    }
    setError("");
    setLoading(true);
    setOffset(0);
    setFilters({
      fromDate: form.fromDate,
      toDate: form.toDate,
      orderNumber: form.orderNumber.trim(),
      cashier: form.cashier.trim(),
      minAmountMinor: minAmountMinor ?? undefined,
      maxAmountMinor: maxAmountMinor ?? undefined,
    });
  }

  function update(name: keyof SearchForm, value: string) {
    setForm((previous) => ({ ...previous, [name]: value }));
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-10">
      <div>
        <Link
          href={workspacePath(current.slug, "/sales")}
          className="inline-flex items-center gap-2 text-sm text-accent hover:underline"
        >
          <ArrowLeft className="h-4 w-4" /> New sale
        </Link>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight">
          Sales history
        </h1>
        <p className="mt-1 text-sm text-muted">
          Find a completed sale and open its original receipt to reprint it.
        </p>
      </div>

      <form
        onSubmit={search}
        className="grid gap-3 rounded-2xl border border-line bg-surface p-4 sm:grid-cols-2 lg:grid-cols-4"
      >
        <label className="text-sm">
          From date
          <input
            className={`${field} mt-1`}
            type="date"
            value={form.fromDate}
            onChange={(event) => update("fromDate", event.target.value)}
          />
        </label>
        <label className="text-sm">
          To date
          <input
            className={`${field} mt-1`}
            type="date"
            value={form.toDate}
            onChange={(event) => update("toDate", event.target.value)}
          />
        </label>
        <label className="text-sm">
          Order number
          <input
            className={`${field} mt-1`}
            placeholder="POS-000123"
            value={form.orderNumber}
            onChange={(event) => update("orderNumber", event.target.value)}
          />
        </label>
        <label className="text-sm">
          Cashier name or ID
          <input
            className={`${field} mt-1`}
            placeholder="Cashier"
            value={form.cashier}
            onChange={(event) => update("cashier", event.target.value)}
          />
        </label>
        <label className="text-sm">
          Minimum amount ({current.currencyCode})
          <input
            className={`${field} mt-1`}
            inputMode="decimal"
            placeholder="0.00"
            value={form.minAmount}
            onChange={(event) => update("minAmount", event.target.value)}
          />
        </label>
        <label className="text-sm">
          Maximum amount ({current.currencyCode})
          <input
            className={`${field} mt-1`}
            inputMode="decimal"
            placeholder="0.00"
            value={form.maxAmount}
            onChange={(event) => update("maxAmount", event.target.value)}
          />
        </label>
        <div className="flex items-end gap-3 sm:col-span-2">
          <button
            type="submit"
            className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright"
          >
            <Search className="h-4 w-4" /> Search sales
          </button>
          <button
            type="button"
            className="px-2 py-2.5 text-sm text-muted hover:text-foreground"
            onClick={() => {
              setForm(emptyForm);
              setFilters({});
              setOffset(0);
              setLoading(true);
            }}
          >
            Clear
          </button>
        </div>
      </form>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}
      <div className="overflow-x-auto rounded-2xl border border-line bg-surface">
        <table className="w-full min-w-[680px] text-left text-sm">
          <thead className="border-b border-line text-muted">
            <tr>
              <th className="p-4 font-medium">Date</th>
              <th className="p-4 font-medium">Order</th>
              <th className="p-4 font-medium">Store</th>
              <th className="p-4 font-medium">Cashier</th>
              <th className="p-4 text-right font-medium">Amount</th>
              <th className="p-4 text-right font-medium">Receipt</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {sales.map((sale) => (
              <tr key={sale.id}>
                <td className="p-4">
                  {new Intl.DateTimeFormat(undefined, {
                    dateStyle: "medium",
                    timeStyle: "short",
                    timeZone: current.timezone,
                  }).format(new Date(sale.createdAt))}
                </td>
                <td className="p-4 font-mono">{sale.orderNumber}</td>
                <td className="p-4">{sale.storeName}</td>
                <td className="p-4">{sale.cashierName || sale.cashierId}</td>
                <td className="p-4 text-right font-semibold">
                  {formatMoney(sale.grandTotalMinor, sale.currencyCode)}
                </td>
                <td className="p-4 text-right">
                  <Link
                    className="text-accent hover:underline"
                    href={workspacePath(current.slug, `/sales/${sale.id}`)}
                  >
                    Open / reprint
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && !error && sales.length === 0 && (
          <p className="p-8 text-center text-sm text-muted">
            No completed sales match these filters.
          </p>
        )}
        {loading && (
          <p className="p-8 text-center text-sm text-muted">Loading sales...</p>
        )}
      </div>
      {(offset > 0 || hasMore) && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-muted">
            Showing {offset + 1}–{offset + sales.length}
          </span>
          <div className="flex gap-3">
            <button
              type="button"
              disabled={loading || offset === 0}
              onClick={() => {
                setLoading(true);
                setOffset(Math.max(0, offset - pageSize));
              }}
              className="rounded-lg border border-line px-3 py-2 disabled:opacity-40"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={loading || !hasMore}
              onClick={() => {
                setLoading(true);
                setOffset(offset + pageSize);
              }}
              className="rounded-lg border border-line px-3 py-2 disabled:opacity-40"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
