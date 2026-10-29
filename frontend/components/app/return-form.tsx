"use client";

import { useRouter } from "next/navigation";
import { useRef, useState, type FormEvent } from "react";
import { Undo2 } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { ShiftPanel } from "@/components/app/shift-panel";
import { Select } from "@/components/ui/select";
import { formatMoney } from "@/lib/money";
import { createReturn, type Sale } from "@/lib/sales-api";
import { lineGross, share } from "@/lib/tax";
import { useCounter } from "@/lib/use-counter";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

/**
 * Takes goods back from one sale. The refund is paid in cash from the drawer
 * of the chosen counter in the store that made the sale, and the goods go
 * back into that store's stock.
 */
export function ReturnForm({ sale }: { sale: Sale }) {
  const { current } = useApp();
  const router = useRouter();
  const counter = useCounter(current.id, sale.storeId);
  const requestKey = useRef<string | null>(null);
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const lines = sale.items
    .map((item) => ({
      item,
      remaining: item.quantity - item.returnedQuantity,
      quantity: quantities[item.variantId] ?? 0,
    }))
    .filter((line) => line.remaining > 0);
  const chosen = lines.filter((line) => line.quantity > 0);
  // What the server will pay back: each line's share of what was charged
  // for it, tax included.
  const refund = chosen.reduce(
    (sum, { item, quantity }) =>
      sum +
      share(
        lineGross(item.lineTotalMinor, item.taxMinor, !!item.taxInclusive),
        item.quantity,
        item.returnedQuantity,
        quantity,
      ),
    0,
  );
  const shiftOpen = !!counter.register?.openSession;
  const canSubmit =
    chosen.length > 0 && reason.trim().length >= 3 && shiftOpen && !busy;

  function setQuantity(variantId: string, value: number, remaining: number) {
    if (!Number.isSafeInteger(value)) return;
    setQuantities((old) => ({
      ...old,
      [variantId]: Math.max(0, Math.min(value, remaining)),
    }));
    requestKey.current = null;
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    setBusy(true);
    setError("");
    requestKey.current ??= crypto.randomUUID();
    try {
      const { return: saved } = await createReturn(current.id, sale.id, {
        storeId: sale.storeId,
        registerId: counter.registerId,
        idempotencyKey: requestKey.current,
        reason: reason.trim(),
        items: chosen.map((line) => ({
          variantId: line.item.variantId,
          quantity: line.quantity,
        })),
      });
      router.push(workspacePath(current.slug, `/sales/returns/${saved.id}`));
    } catch (cause) {
      // Keep the key: if the refund committed before the network failed, a
      // retry returns it instead of paying twice.
      setError(
        cause instanceof Error ? cause.message : "Could not record the return.",
      );
      setBusy(false);
    }
  }

  if (lines.length === 0) {
    return (
      <p className="text-sm text-muted">
        Every item on this sale has already been returned.
      </p>
    );
  }

  return (
    <form
      onSubmit={(event) => void submit(event)}
      className="space-y-5 rounded-2xl border border-line bg-surface p-5"
    >
      <div>
        <h2 className="flex items-center gap-2 text-lg font-semibold">
          <Undo2 className="h-5 w-5 text-accent-ink" /> Return items
        </h2>
        <p className="mt-1 text-sm text-muted">
          Choose what the customer is bringing back. The items go back into{" "}
          {sale.storeName} stock and the refund is paid in cash.
        </p>
      </div>

      <div className="divide-y divide-line border-y border-line">
        {lines.map(({ item, remaining, quantity }) => (
          <div
            key={item.variantId}
            className="flex flex-wrap items-center justify-between gap-3 py-3 text-sm"
          >
            <div className="min-w-0">
              <p className="font-medium">
                {item.productName}
                {item.variantName !== "Standard" && ` · ${item.variantName}`}
              </p>
              <p className="mt-1 text-xs text-muted">
                {item.sku} ·{" "}
                {formatMoney(item.unitPriceMinor, sale.currencyCode)} each ·{" "}
                {remaining} of {item.quantity} can be returned
              </p>
            </div>
            <label className="flex items-center gap-2 text-xs text-muted">
              Return
              <input
                type="number"
                min={0}
                max={remaining}
                value={quantity}
                aria-label={`Units of ${item.sku} to return`}
                onChange={(event) =>
                  setQuantity(
                    item.variantId,
                    Number(event.target.value),
                    remaining,
                  )
                }
                className="w-20 rounded-lg border border-line bg-canvas px-2 py-1.5 text-center text-sm text-foreground outline-none focus:border-accent"
              />
            </label>
          </div>
        ))}
      </div>

      <label className="block text-sm font-medium">
        Reason for the return
        <textarea
          className={`${field} mt-2 min-h-20`}
          maxLength={500}
          placeholder="e.g. Wrong size, customer exchanged for a larger one"
          value={reason}
          onChange={(event) => {
            setReason(event.target.value);
            requestKey.current = null;
          }}
        />
        <span className="mt-1 block text-xs text-muted">
          Required. Kept with the return and shown on the refund receipt.
        </span>
      </label>

      {counter.registers.length > 0 && (
        <label className="block max-w-xs text-sm font-medium">
          Refund from counter
          <Select
            ariaLabel="Refund from counter"
            className={`${field} mt-2`}
            value={counter.registerId}
            onValueChange={(id) => {
              counter.select(id);
              requestKey.current = null;
            }}
            disabled={busy}
            options={counter.registers.map((register) => ({
              value: register.id,
              label: `${register.name}${register.openSession ? " (open)" : ""}`,
            }))}
          />
        </label>
      )}
      {counter.error && (
        <p role="alert" className="text-sm text-danger-ink">
          {counter.error}
        </p>
      )}
      {counter.register && !shiftOpen && (
        <ShiftPanel
          storeId={sale.storeId}
          register={counter.register}
          onChanged={() => void counter.refresh()}
        />
      )}

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-line pt-4">
        <p className="text-sm">
          Cash refund{" "}
          <span className="text-lg font-semibold">
            {formatMoney(refund, sale.currencyCode)}
          </span>
        </p>
        <button
          type="submit"
          disabled={!canSubmit}
          className="rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50"
        >
          {busy ? "Recording return…" : "Confirm return and refund"}
        </button>
      </div>
    </form>
  );
}
