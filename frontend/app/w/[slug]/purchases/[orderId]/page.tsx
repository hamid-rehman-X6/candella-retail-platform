"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { ArrowLeft } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import {
  approvePurchase,
  getPurchase,
  receivePurchase,
  type PurchaseOrder,
} from "@/lib/purchases-api";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import { toast } from "@/lib/toast";

const field =
  "w-24 rounded-xl border border-line bg-surface px-3 py-2 text-sm text-foreground outline-none focus:border-accent";
const button =
  "cursor-pointer rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50";

export default function PurchaseDetailPage() {
  const { current } = useApp();
  const { orderId } = useParams<{ orderId: string }>();
  const [order, setOrder] = useState<PurchaseOrder | null>(null);
  const [quantity, setQuantity] = useState<Record<string, string>>({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const requestKey = useRef<string | null>(null);
  const manager = current.role === "owner" || current.role === "admin";

  useEffect(() => {
    if (!manager) return;
    let cancelled = false;
    void getPurchase(current.id, orderId)
      .then(({ order }) => {
        if (cancelled) return;
        setOrder(order);
        setQuantity(
          Object.fromEntries(
            order.items.map((item) => [
              item.variantId,
              String(item.quantity - item.receivedQuantity),
            ]),
          ),
        );
        setLoading(false);
      })
      .catch((e) => {
        if (!cancelled) {
          setError(
            e instanceof Error ? e.message : "Could not open purchase order.",
          );
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, orderId, manager]);

  async function approve() {
    if (!order || busy) return;
    setBusy(true);
    setError("");
    try {
      const result = await approvePurchase(current.id, order.id);
      setOrder(result.order);
      toast.success("Purchase order approved");
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "Could not approve purchase order.",
      );
    } finally {
      setBusy(false);
    }
  }
  const receivingLines =
    order?.items.map((item) => ({
      item,
      count: Number(quantity[item.variantId] ?? "0"),
    })) ?? [];
  const canReceive =
    receivingLines.some(({ count }) => count > 0) &&
    receivingLines.every(
      ({ item, count }) =>
        Number.isSafeInteger(count) &&
        count >= 0 &&
        count <= item.quantity - item.receivedQuantity,
    );
  async function receive(e: FormEvent) {
    e.preventDefault();
    if (!order || !canReceive || busy) return;
    setBusy(true);
    setError("");
    requestKey.current ??= crypto.randomUUID();
    try {
      const result = await receivePurchase(current.id, order.id, {
        idempotencyKey: requestKey.current,
        items: receivingLines
          .filter(({ count }) => count > 0)
          .map(({ item, count }) => ({
            variantId: item.variantId,
            quantity: count,
          })),
      });
      setOrder(result.order);
      setQuantity(
        Object.fromEntries(
          result.order.items.map((item) => [
            item.variantId,
            String(item.quantity - item.receivedQuantity),
          ]),
        ),
      );
      requestKey.current = null;
      toast.success("Delivery received and stock updated");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not receive delivery.");
    } finally {
      setBusy(false);
    }
  }

  if (!manager)
    return (
      <div className="mx-auto max-w-5xl p-6 text-sm text-muted">
        Ask a workspace owner for purchase order access.
      </div>
    );
  return (
    <div className="mx-auto max-w-4xl space-y-6 pb-12">
      <Link
        href={workspacePath(current.slug, "/purchases")}
        className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline"
      >
        <ArrowLeft className="h-4 w-4" /> Purchases
      </Link>
      {loading && <p className="text-sm text-muted">Loading purchase order…</p>}
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}
      {order && (
        <>
          <header className="rounded-2xl border border-line bg-surface p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <p className="text-xs font-semibold tracking-wider text-accent-ink uppercase">
                  Purchase order
                </p>
                <h1 className="mt-1 text-3xl font-semibold">{order.number}</h1>
                <p className="mt-2 text-sm text-muted">
                  {order.supplierName} · {order.storeName} ·{" "}
                  {formatDateTime(order.createdAt, current.timezone)}
                </p>
              </div>
              <span className="rounded-full bg-accent/10 px-3 py-1 text-sm font-medium text-accent-ink">
                {order.status.replaceAll("_", " ")}
              </span>
            </div>
            <div className="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-line pt-5">
              <p className="font-semibold">
                Order total:{" "}
                {formatMoney(order.totalCostMinor, order.currencyCode)}
              </p>
              {order.status === "draft" && (
                <button
                  type="button"
                  onClick={() => void approve()}
                  disabled={busy}
                  className={button}
                >
                  {busy ? "Approving…" : "Approve order"}
                </button>
              )}
            </div>
            {order.status === "draft" && (
              <p className="mt-3 text-xs text-muted">
                This draft has not changed stock. Review quantities and costs
                before approval.
              </p>
            )}
          </header>
          <section className="overflow-hidden rounded-2xl border border-line bg-surface">
            <h2 className="border-b border-line p-5 text-lg font-semibold">
              Ordered products
            </h2>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[560px] text-left text-sm">
                <thead className="border-b border-line text-muted">
                  <tr>
                    <th className="p-4 font-medium">Product / SKU</th>
                    <th className="p-4 font-medium">Ordered</th>
                    <th className="p-4 font-medium">Received</th>
                    <th className="p-4 font-medium">Unit cost</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line">
                  {order.items.map((item) => (
                    <tr key={item.variantId}>
                      <td className="p-4">
                        <span className="font-medium">
                          {item.productName} · {item.variantName}
                        </span>
                        <span className="block font-mono text-xs text-muted">
                          {item.sku}
                        </span>
                      </td>
                      <td className="p-4">{item.quantity}</td>
                      <td className="p-4">{item.receivedQuantity}</td>
                      <td className="p-4">
                        {formatMoney(item.unitCostMinor, order.currencyCode)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          {(order.status === "approved" ||
            order.status === "partially_received") && (
            <form
              onSubmit={(e) => void receive(e)}
              className="space-y-4 rounded-2xl border border-line bg-surface p-5"
            >
              <div>
                <h2 className="text-lg font-semibold">Receive delivery</h2>
                <p className="mt-1 text-sm text-muted">
                  Enter units actually delivered. Leave a line at zero to
                  receive it later.
                </p>
              </div>
              {order.items
                .filter((item) => item.receivedQuantity < item.quantity)
                .map((item) => (
                  <label
                    key={item.variantId}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line p-3 text-sm"
                  >
                    <span>
                      <span className="block font-medium">
                        {item.productName} · {item.variantName}
                      </span>
                      <span className="text-xs text-muted">
                        {item.quantity - item.receivedQuantity} remaining ·{" "}
                        {item.sku}
                      </span>
                    </span>
                    <span className="text-xs text-muted">
                      Received now
                      <input
                        className={`${field} mt-1 block`}
                        type="number"
                        min="0"
                        max={item.quantity - item.receivedQuantity}
                        step="1"
                        value={quantity[item.variantId] ?? "0"}
                        onChange={(e) => {
                          setQuantity((old) => ({
                            ...old,
                            [item.variantId]: e.target.value,
                          }));
                          requestKey.current = null;
                        }}
                        disabled={busy}
                      />
                    </span>
                  </label>
                ))}
              <button
                type="submit"
                disabled={!canReceive || busy}
                className={button}
              >
                {busy ? "Recording delivery…" : "Confirm received stock"}
              </button>
              <p className="text-xs text-muted">
                Confirm once the units are counted. Stock and the movement
                history update together.
              </p>
            </form>
          )}
          <section className="space-y-3 rounded-2xl border border-line bg-surface p-5">
            <h2 className="text-lg font-semibold">Delivery history</h2>
            {order.receipts.length === 0 ? (
              <p className="text-sm text-muted">No deliveries received yet.</p>
            ) : (
              order.receipts.map((receipt) => (
                <div
                  key={receipt.id}
                  className="rounded-xl border border-line p-3 text-sm"
                >
                  <div className="flex flex-wrap justify-between gap-2">
                    <span className="font-medium">
                      {formatDateTime(receipt.createdAt, current.timezone)}
                    </span>
                    <span>
                      {formatMoney(receipt.totalCostMinor, order.currencyCode)}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-muted">
                    {receipt.items
                      .map(
                        (line) =>
                          `${order.items.find((item) => item.variantId === line.variantId)?.sku ?? line.variantId}: ${line.quantity}`,
                      )
                      .join(" · ")}
                  </p>
                </div>
              ))
            )}
            {order.receipts.length > 0 && (
              <Link
                href={workspacePath(current.slug, "/inventory")}
                className="inline-block text-sm text-accent-ink hover:underline"
              >
                View stock movements
              </Link>
            )}
          </section>
        </>
      )}
    </div>
  );
}
