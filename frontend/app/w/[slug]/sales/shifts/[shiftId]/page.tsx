"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ArrowLeft, Lock } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { VarianceBadge } from "@/components/app/variance-badge";
import { ApiError } from "@/lib/api";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney, parseMoney } from "@/lib/money";
import { closeShift, getShift, type Shift } from "@/lib/register-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex justify-between gap-4 py-2">
      <dt className="text-muted">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}

export default function ShiftPage() {
  const { current, user } = useApp();
  const { shiftId } = useParams<{ shiftId: string }>();
  const [shift, setShift] = useState<Shift | null>(null);
  const [counted, setCounted] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    const data = await getShift(current.id, shiftId);
    setShift(data.shift);
    return data.shift;
  }, [current.id, shiftId]);

  useEffect(() => {
    let cancelled = false;
    void getShift(current.id, shiftId)
      .then((data) => {
        if (!cancelled) setShift(data.shift);
      })
      .catch((cause) => {
        if (!cancelled)
          setError(
            cause instanceof Error ? cause.message : "Could not open shift.",
          );
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, shiftId]);

  const canClose =
    !!shift &&
    shift.status === "open" &&
    (shift.openedBy === user.id ||
      current.role === "owner" ||
      current.role === "admin");
  const countedMinor = counted.trim()
    ? parseMoney(counted, current.currencyCode)
    : null;
  const variance =
    shift && countedMinor !== null
      ? countedMinor - shift.expectedCashMinor
      : null;
  const needsReason = variance !== null && variance !== 0;
  const canSubmit =
    canClose &&
    countedMinor !== null &&
    (!needsReason || reason.trim().length >= 3) &&
    !busy;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!shift || !canSubmit || countedMinor === null) return;
    setBusy(true);
    setError("");
    try {
      const data = await closeShift(current.id, shift.id, {
        countedCashMinor: countedMinor,
        expectedCashMinor: shift.expectedCashMinor,
        varianceReason: reason.trim(),
      });
      setShift(data.shift);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not close the shift.",
      );
      // A sale landed meanwhile, or someone else closed it: show the shift now.
      if (
        cause instanceof ApiError &&
        (cause.code === "shift_changed" || cause.code === "shift_closed")
      )
        void load().catch(() => undefined);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 pb-12">
      <div className="flex flex-wrap gap-4">
        <Link
          href={workspacePath(current.slug, "/sales/shifts")}
          className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline"
        >
          <ArrowLeft className="h-4 w-4" /> Shifts
        </Link>
        <Link
          href={workspacePath(current.slug, "/sales")}
          className="text-sm text-accent-ink hover:underline"
        >
          Point of sale
        </Link>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}
      {!shift && !error && <p className="text-sm text-muted">Loading shift…</p>}

      {shift && (
        <>
          <div>
            <p className="text-xs font-semibold tracking-[0.22em] text-accent-ink uppercase">
              {shift.status === "open" ? "Open shift" : "Closed shift"}
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight">
              {shift.registerName} · {shift.storeName}
            </h1>
            <p className="mt-1 text-sm text-muted">
              Opened by {shift.openerName} on{" "}
              {formatDateTime(shift.openedAt, current.timezone)}
              {shift.closedAt &&
                ` · closed by ${shift.closerName} on ${formatDateTime(shift.closedAt, current.timezone)}`}
            </p>
          </div>

          <section className="rounded-2xl border border-line bg-surface p-5">
            <h2 className="text-lg font-semibold">Drawer</h2>
            <dl className="mt-2 divide-y divide-line text-sm">
              <Row
                label="Opening cash"
                value={formatMoney(shift.openingCashMinor, shift.currencyCode)}
              />
              <Row
                label={`Cash sales (${shift.salesCount})`}
                value={formatMoney(shift.cashSalesMinor, shift.currencyCode)}
              />
              {shift.refundsCount > 0 && (
                <Row
                  label={`Cash refunds (${shift.refundsCount})`}
                  value={`− ${formatMoney(shift.cashRefundsMinor, shift.currencyCode)}`}
                />
              )}
              <Row
                label="Expected in drawer"
                value={formatMoney(shift.expectedCashMinor, shift.currencyCode)}
              />
              {shift.countedCashMinor !== null && (
                <Row
                  label="Counted at closing"
                  value={formatMoney(
                    shift.countedCashMinor,
                    shift.currencyCode,
                  )}
                />
              )}
              {shift.varianceMinor !== null && (
                <Row
                  label="Difference"
                  value={
                    <VarianceBadge
                      varianceMinor={shift.varianceMinor}
                      currencyCode={shift.currencyCode}
                    />
                  }
                />
              )}
              {shift.varianceReason && (
                <Row label="Reason" value={shift.varianceReason} />
              )}
            </dl>
            {shift.status === "closed" && (
              <p className="mt-4 flex items-center gap-2 text-xs text-muted">
                <Lock className="h-3.5 w-3.5" /> A closed shift is final and
                cannot be edited.
              </p>
            )}
          </section>

          {canClose && (
            <form
              onSubmit={(event) => void submit(event)}
              className="space-y-4 rounded-2xl border border-line bg-surface p-5"
            >
              <div>
                <h2 className="text-lg font-semibold">Close shift</h2>
                <p className="mt-1 text-sm text-muted">
                  Count the cash in the drawer and enter the total. After
                  closing, no more sales can go into this shift.
                </p>
              </div>
              <label className="block text-sm font-medium">
                Counted cash ({current.currencyCode})
                <input
                  className={`${field} mt-2`}
                  inputMode="decimal"
                  placeholder="0"
                  value={counted}
                  onChange={(event) => setCounted(event.target.value)}
                />
              </label>
              {counted.trim() && countedMinor === null && (
                <p className="text-xs text-danger-ink">
                  Enter the counted cash as an amount, e.g. 12500.
                </p>
              )}
              {variance !== null && (
                <p className="text-sm">
                  Expected{" "}
                  {formatMoney(shift.expectedCashMinor, shift.currencyCode)} ·{" "}
                  <VarianceBadge
                    varianceMinor={variance}
                    currencyCode={shift.currencyCode}
                  />
                </p>
              )}
              {needsReason && (
                <label className="block text-sm font-medium">
                  Reason for the difference
                  <textarea
                    className={`${field} mt-2 min-h-20`}
                    maxLength={500}
                    placeholder="e.g. Gave Rs 50 extra change on one sale"
                    value={reason}
                    onChange={(event) => setReason(event.target.value)}
                  />
                  <span className="mt-1 block text-xs text-muted">
                    Required when counted cash does not match. Kept with the
                    shift record.
                  </span>
                </label>
              )}
              <button
                type="submit"
                disabled={!canSubmit}
                className="w-full rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50"
              >
                {busy ? "Closing…" : "Close shift"}
              </button>
            </form>
          )}
        </>
      )}
    </div>
  );
}
