"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { ArrowLeft, Printer, Undo2 } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { ReturnForm } from "@/components/app/return-form";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import { formatRate } from "@/lib/tax";
import { getSale, type Sale } from "@/lib/sales-api";

export default function ReceiptPage() {
  const { current } = useApp();
  const { orderId } = useParams<{ orderId: string }>();
  const [sale, setSale] = useState<Sale | null>(null);
  const [error, setError] = useState("");
  const [returning, setReturning] = useState(false);
  const canRefund = current.role === "owner" || current.role === "admin";
  const returnable =
    !!sale && sale.items.some((item) => item.returnedQuantity < item.quantity);

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
          <div className="flex flex-wrap gap-3">
            {canRefund && returnable && (
              <button
                type="button"
                onClick={() => setReturning((open) => !open)}
                className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2 text-sm font-medium hover:border-accent/50"
              >
                <Undo2 className="h-4 w-4" />{" "}
                {returning ? "Cancel return" : "Return items"}
              </button>
            )}
            <button
              type="button"
              onClick={() => window.print()}
              className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2 text-sm font-semibold text-black"
            >
              <Printer className="h-4 w-4" /> Print receipt
            </button>
          </div>
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
            <p className="mt-1 text-sm">
              Store: {sale.storeName}
              {sale.registerName && ` · ${sale.registerName}`}
            </p>
            <p className="mt-2 font-mono text-sm">{sale.orderNumber}</p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              {formatDateTime(sale.createdAt, current.timezone)}
            </p>
            <p className="mt-1 text-xs text-muted print:text-gray-600">
              Cashier: {sale.cashierName || sale.cashierId}
            </p>
            {sale.customerName && (
              <p className="mt-1 text-xs text-muted print:text-gray-600">
                Customer: {sale.customerName}
              </p>
            )}
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
                  {item.discountMinor > 0 && (
                    <p className="mt-1 text-xs text-success-ink print:text-black">
                      {item.promotionName ?? "Discount"}: −
                      {formatMoney(item.discountMinor, sale.currencyCode)}
                    </p>
                  )}
                  {item.taxName && item.taxRateBps !== null && (
                    <p className="mt-1 text-xs text-muted print:text-gray-600">
                      {item.taxName} ({formatRate(item.taxRateBps)}
                      {item.taxInclusive ? ", included" : ", added"}):{" "}
                      {formatMoney(item.taxMinor, sale.currencyCode)}
                    </p>
                  )}
                  {item.returnedQuantity > 0 && (
                    <p className="mt-1 text-xs font-medium text-danger-ink print:text-black">
                      {item.returnedQuantity} returned
                    </p>
                  )}
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
            {sale.discountMinor > 0 && (
              <div className="flex justify-between gap-4">
                <dt>
                  {sale.discountName ?? "Item discounts"}
                  {sale.discountReason && (
                    <span className="block text-xs text-muted print:text-gray-600">
                      {sale.discountReason}
                    </span>
                  )}
                </dt>
                <dd>−{formatMoney(sale.discountMinor, sale.currencyCode)}</dd>
              </div>
            )}
            {sale.taxMinor > 0 && (
              <div className="flex justify-between">
                <dt>
                  Tax
                  {sale.grandTotalMinor ===
                    sale.subtotalMinor - sale.discountMinor &&
                    " (included in prices)"}
                </dt>
                <dd>{formatMoney(sale.taxMinor, sale.currencyCode)}</dd>
              </div>
            )}
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
          {sale.returns.length > 0 && (
            <section className="mt-6 border-t border-line pt-5 text-sm print:border-gray-300">
              <h2 className="font-semibold">Returns</h2>
              <ul className="mt-2 space-y-2">
                {sale.returns.map((item) => (
                  <li key={item.id} className="flex justify-between gap-4">
                    <span>
                      <Link
                        href={workspacePath(
                          current.slug,
                          `/sales/returns/${item.id}`,
                        )}
                        className="font-mono text-accent-ink hover:underline print:text-black print:no-underline"
                      >
                        {item.returnNumber}
                      </Link>{" "}
                      <span className="text-muted print:text-gray-600">
                        · {formatDateTime(item.createdAt, current.timezone)} ·{" "}
                        {item.reason}
                      </span>
                    </span>
                    <span className="shrink-0">
                      −{formatMoney(item.refundMinor, sale.currencyCode)}
                    </span>
                  </li>
                ))}
              </ul>
              <div className="mt-3 flex justify-between font-semibold">
                <span>Net after refunds</span>
                <span>
                  {formatMoney(
                    sale.grandTotalMinor - sale.refundedMinor,
                    sale.currencyCode,
                  )}
                </span>
              </div>
            </section>
          )}
          <p className="mt-8 text-center text-xs text-muted print:text-gray-600">
            Thank you for shopping with us.
          </p>
        </article>
      )}
      {sale && canRefund && returning && (
        <div className="mt-6 print:hidden">
          <ReturnForm sale={sale} />
        </div>
      )}
    </div>
  );
}
