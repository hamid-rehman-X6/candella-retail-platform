"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ArrowLeft, Printer } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import { getReturn, type SaleReturn } from "@/lib/sales-api";

export default function RefundReceiptPage() {
  const { current } = useApp();
  const { returnId } = useParams<{ returnId: string }>();
  const [refund, setRefund] = useState<SaleReturn | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;
    void getReturn(current.id, returnId)
      .then(({ return: loaded }) => {
        if (!cancelled) setRefund(loaded);
      })
      .catch((e) => {
        if (!cancelled)
          setError(e instanceof Error ? e.message : "Could not open return.");
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, returnId]);

  return (
    <div className="mx-auto max-w-2xl pb-12">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 print:hidden">
        <div className="flex flex-wrap gap-4">
          {refund && (
            <Link
              href={workspacePath(current.slug, `/sales/${refund.orderId}`)}
              className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline"
            >
              <ArrowLeft className="h-4 w-4" /> Sale {refund.orderNumber}
            </Link>
          )}
          <Link
            href={workspacePath(current.slug, "/sales")}
            className="text-sm text-accent-ink hover:underline"
          >
            Point of sale
          </Link>
        </div>
        {refund && (
          <button
            type="button"
            onClick={() => window.print()}
            className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-black"
          >
            <Printer className="h-4 w-4" /> Print refund receipt
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
      {!refund && !error && (
        <p className="text-sm text-muted">Loading return…</p>
      )}
      {refund && (
        <article className="rounded-2xl border border-line bg-surface p-6 text-foreground sm:p-9 print:rounded-none print:border-0 print:bg-white print:p-0 print:text-black">
          <header className="border-b border-line pb-6 text-center print:border-gray-300">
            <p className="text-xs font-semibold tracking-[0.2em] text-accent-ink uppercase print:text-black">
              Refund receipt
            </p>
            <h1 className="mt-2 text-2xl font-semibold">{current.name}</h1>
            <p className="mt-1 text-sm">
              Store: {refund.storeName} · {refund.registerName}
            </p>
            <p className="mt-2 font-mono text-sm">{refund.returnNumber}</p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              For sale {refund.orderNumber} ·{" "}
              {formatDateTime(refund.createdAt, current.timezone)}
            </p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              Processed by: {refund.processorName}
            </p>
          </header>
          <div className="divide-y divide-line print:divide-gray-300">
            {refund.items.map((item) => (
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
                    {formatMoney(item.unitPriceMinor, refund.currencyCode)}
                  </p>
                </div>
                <span className="shrink-0 font-medium">
                  {formatMoney(item.lineRefundMinor, refund.currencyCode)}
                </span>
              </div>
            ))}
          </div>
          <dl className="space-y-2 border-t border-line pt-5 text-sm print:border-gray-300">
            <div className="flex justify-between text-lg font-semibold">
              <dt>Cash refunded</dt>
              <dd>{formatMoney(refund.refundMinor, refund.currencyCode)}</dd>
            </div>
            {refund.taxMinor > 0 && (
              <div className="flex justify-between">
                <dt>Of which tax</dt>
                <dd>{formatMoney(refund.taxMinor, refund.currencyCode)}</dd>
              </div>
            )}
            <div className="flex justify-between gap-4">
              <dt>Reason</dt>
              <dd className="text-right">{refund.reason}</dd>
            </div>
          </dl>
        </article>
      )}
    </div>
  );
}
