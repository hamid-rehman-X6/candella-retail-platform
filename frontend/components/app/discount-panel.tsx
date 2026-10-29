"use client";

import { BadgePercent } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { Select } from "@/components/ui/select";
import { formatMoney } from "@/lib/money";
import type { Promotion } from "@/lib/promotions-api";
import { formatRate } from "@/lib/tax";

export type OrderDiscount =
  | { mode: "none" }
  | { mode: "promotion"; promotionId: string }
  | { mode: "manual"; percent: string; reason: string };

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2 text-sm text-foreground outline-none focus:border-accent";

/** "10", "12.5" or "10%" to basis points; null when not a percentage. */
export function parsePercent(value: string): number | null {
  const text = value.trim().replace(/%$/, "").trim();
  if (!/^\d{1,3}(\.\d{1,2})?$/.test(text)) return null;
  const bps = Math.round(Number(text) * 100);
  return bps >= 1 && bps <= 10000 ? bps : null;
}

export function promotionLabel(p: Promotion, currency: string): string {
  const off =
    p.kind === "percent"
      ? formatRate(p.percentBps ?? 0)
      : formatMoney(p.amountMinor ?? 0, currency);
  const each = p.appliesTo === "item" && p.kind === "amount" ? " each" : "";
  const minimum =
    p.minSubtotalMinor > 0
      ? `, bills of ${formatMoney(p.minSubtotalMinor, currency)}+`
      : "";
  return `${p.name} (${off} off${each}${minimum})`;
}

/**
 * The bill-level discount: an approved promotion, or a manual percentage with
 * a reason. Cashiers see the limit the owner set; anything above it needs a
 * manager to ring the sale.
 */
export function DiscountPanel({
  value,
  onChange,
  promotions,
  cashierLimitBps,
  blockedByItems,
  disabled,
}: {
  value: OrderDiscount;
  onChange: (value: OrderDiscount) => void;
  promotions: Promotion[];
  cashierLimitBps: number;
  blockedByItems: boolean;
  disabled: boolean;
}) {
  const { current } = useApp();
  const manager = current.role === "owner" || current.role === "admin";
  const canManual = manager || cashierLimitBps > 0;
  const percentBps =
    value.mode === "manual" ? parsePercent(value.percent) : null;
  const overLimit =
    value.mode === "manual" &&
    !manager &&
    percentBps !== null &&
    percentBps > cashierLimitBps;

  if (promotions.length === 0 && !canManual) return null;

  return (
    <div className="space-y-2 rounded-xl border border-line p-3">
      <p className="flex items-center gap-2 text-sm font-medium">
        <BadgePercent className="h-4 w-4 text-accent-ink" /> Bill discount
      </p>
      {blockedByItems ? (
        <p className="text-xs text-muted">
          An item promotion is applied. Remove it to give a discount on the
          whole bill.
        </p>
      ) : (
        <>
          <Select
            ariaLabel="Bill discount"
            className={field}
            value={
              value.mode === "promotion"
                ? value.promotionId
                : value.mode === "manual"
                  ? "manual"
                  : ""
            }
            disabled={disabled}
            onValueChange={(choice) =>
              onChange(
                choice === ""
                  ? { mode: "none" }
                  : choice === "manual"
                    ? { mode: "manual", percent: "", reason: "" }
                    : { mode: "promotion", promotionId: choice },
              )
            }
            options={[
              { value: "", label: "No discount" },
              ...promotions.map((p) => ({
                value: p.id,
                label: promotionLabel(p, current.currencyCode),
              })),
              ...(canManual
                ? [{ value: "manual", label: "Manual discount (%)" }]
                : []),
            ]}
          />
          {value.mode === "manual" && (
            <div className="grid gap-2 sm:grid-cols-[96px_minmax(0,1fr)]">
              <input
                aria-label="Discount percentage"
                className={field}
                inputMode="decimal"
                placeholder="%"
                value={value.percent}
                disabled={disabled}
                onChange={(e) =>
                  onChange({ ...value, percent: e.target.value })
                }
              />
              <input
                aria-label="Reason for the discount"
                className={field}
                maxLength={300}
                placeholder="Reason, e.g. regular customer"
                value={value.reason}
                disabled={disabled}
                onChange={(e) => onChange({ ...value, reason: e.target.value })}
              />
            </div>
          )}
          {value.mode === "manual" && !manager && (
            <p
              className={`text-xs ${overLimit ? "text-danger-ink" : "text-muted"}`}
            >
              You can give up to {formatRate(cashierLimitBps)} off. Ask an owner
              or admin for more.
            </p>
          )}
        </>
      )}
    </div>
  );
}
