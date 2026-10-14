"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ArrowLeft, Printer } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { formatMoney } from "@/lib/catalog-api";
import { getSale, type Sale } from "@/lib/sales-api";

export default function ReceiptPage() {
  const { current } = useApp();
  const { orderId } = useParams<{ orderId: string }>();
  const [sale, setSale] = useState<Sale | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    void getSale(current.id, orderId)
      .then(({ sale }) => {
        if (!cancelled) setSale(sale);
      })
      .catch((e) => {
        if (!cancelled)
          setError(e instanceof Error ? e.message : "Could not open receipt.");
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, orderId]);

  return (
    <div className="mx-auto max-w-2xl pb-12">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <div className="flex flex-wrap gap-4">
          <Link
            href={workspacePath(current.slug, "/sales/history")}
            className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline"
          >
            <ArrowLeft className="h-4 w-4" /> Sales history
          </Link>
          <Link
            href={workspacePath(current.slug, "/sales")}
            className="text-sm text-accent-ink hover:underline"
          >
            New sale
          </Link>
        </div>
        {sale && (
          <button
            type="button"
            onClick={() => window.print()}
            className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-black"
          >
            <Printer className="h-4 w-4" /> Print receipt
          </button>
        )}
      </div>
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 p-4 text-danger-ink"
        >
          {error}
        </p>
      )}
      {!sale && !error && (
        <p className="text-sm text-muted">Loading receipt…</p>
      )}
      {sale && (
        <article className="rounded-2xl border border-line bg-surface p-6 text-foreground sm:p-9 print:rounded-none print:border-0 print:bg-white print:p-0 print:text-black">
          <header className="border-b border-line pb-6 text-center print:border-gray-300">
            <p className="text-xs font-semibold tracking-[0.2em] text-accent-ink uppercase print:text-black">
              Cash receipt
            </p>
            <h1 className="mt-2 text-2xl font-semibold">{current.name}</h1>
            <p className="mt-1 text-sm">Store: {sale.storeName}</p>
            <p className="mt-2 font-mono text-sm">{sale.orderNumber}</p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              {new Intl.DateTimeFormat(undefined, {
                dateStyle: "medium",
                timeStyle: "short",
                timeZone: current.timezone,
              }).format(new Date(sale.createdAt))}
            </p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              Cashier: {sale.cashierName || sale.cashierId}
            </p>
          </header>
          <div className="divide-y divide-line print:divide-gray-300">
            {sale.items.map((item) => (
              <div
                key={item.variantId}
                className="flex justify-between gap-4 py-4 text-sm"
              >
                <div>
                  <p className="font-medium">
                    {item.productName}
                    {item.variantName !== "Standard" &&
                      ` · ${item.variantName}`}
                  </p>
                  <p className="mt-1 text-xs text-muted print:text-gray-600">
                    {item.sku} · {item.quantity} ×{" "}
                    {formatMoney(item.unitPriceMinor, sale.currencyCode)}
                  </p>
                </div>
                <span className="shrink-0 font-medium">
                  {formatMoney(item.lineTotalMinor, sale.currencyCode)}
                </span>
              </div>
            ))}
          </div>
          <dl className="space-y-2 border-t border-line pt-5 text-sm print:border-gray-300">
            <div className="flex justify-between">
              <dt>Subtotal</dt>
              <dd>{formatMoney(sale.subtotalMinor, sale.currencyCode)}</dd>
            </div>
            <div className="flex justify-between text-lg font-semibold">
              <dt>Total</dt>
              <dd>{formatMoney(sale.grandTotalMinor, sale.currencyCode)}</dd>
            </div>
            <div className="flex justify-between">
              <dt>Cash received</dt>
              <dd>{formatMoney(sale.cashReceivedMinor, sale.currencyCode)}</dd>
            </div>
            <div className="flex justify-between">
              <dt>Change given</dt>
              <dd>{formatMoney(sale.changeMinor, sale.currencyCode)}</dd>
            </div>
          </dl>
          <p className="mt-8 text-center text-xs text-muted print:text-gray-600">
            Thank you for shopping with us.
          </p>
        </article>
      )}
    </div>
  );
}
