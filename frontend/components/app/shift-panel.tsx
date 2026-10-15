"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";
import { LockOpen, Wallet } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { formatDateTime } from "@/lib/datetime";
import { parseMoney } from "@/lib/money";
import { openShift, type Register } from "@/lib/register-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

/**
 * Shows whose shift the selected counter is on, or lets the cashier open one.
 * Cash can only be taken while a shift is open, so this sits above the cart.
 */
export function ShiftPanel({
  storeId,
  register,
  onChanged,
}: {
  storeId: string;
  register: Register;
  onChanged: () => void;
}) {
  const { current, user } = useApp();
  const [opening, setOpening] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const session = register.openSession;
  const canSeeDrawer =
    !!session &&
    (session.openedBy === user.id ||
      current.role === "owner" ||
      current.role === "admin");

  if (session) {
    return (
      <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm">
        <p className="flex items-center gap-2">
          <LockOpen className="h-4 w-4 text-success-ink" />
          <span>
            <span className="font-medium">{register.name}</span> is open —{" "}
            {session.openedBy === user.id ? "your shift" : session.openerName},
            since {formatDateTime(session.openedAt, current.timezone)}
          </span>
        </p>
        {canSeeDrawer && (
          <Link
            href={workspacePath(current.slug, `/sales/shifts/${session.id}`)}
            className="font-medium text-accent-ink hover:underline"
          >
            View drawer &amp; close shift
          </Link>
        )}
      </div>
    );
  }

  const openingMinor = opening.trim()
    ? parseMoney(opening, current.currencyCode)
    : 0;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (openingMinor === null || busy) return;
    setBusy(true);
    setError("");
    try {
      await openShift(current.id, {
        storeId,
        registerId: register.id,
        openingCashMinor: openingMinor,
      });
      setOpening("");
      onChanged();
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not open the shift.",
      );
      // Someone else may have just opened it; show who.
      onChanged();
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={(event) => void submit(event)}
      className="rounded-2xl border border-amber-500/30 bg-amber-500/10 p-4"
    >
      <p className="flex items-center gap-2 text-sm font-medium">
        <Wallet className="h-4 w-4 text-accent-ink" />
        Open a shift on {register.name} to take cash
      </p>
      <p className="mt-1 text-xs text-muted">
        Count the cash already in the drawer (the float). It is the starting
        point when the shift is closed.
      </p>
      <div className="mt-3 flex flex-wrap items-end gap-3">
        <label className="min-w-48 flex-1 text-sm">
          Opening cash ({current.currencyCode})
          <input
            className={`${field} mt-1`}
            inputMode="decimal"
            placeholder="0"
            value={opening}
            onChange={(event) => setOpening(event.target.value)}
          />
        </label>
        <button
          type="submit"
          disabled={busy || openingMinor === null}
          className="rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50"
        >
          {busy ? "Opening…" : "Open shift"}
        </button>
      </div>
      {openingMinor === null && (
        <p className="mt-2 text-xs text-danger-ink">
          Enter the opening cash as an amount, e.g. 5000.
        </p>
      )}
      {error && (
        <p role="alert" className="mt-2 text-sm text-danger-ink">
          {error}
        </p>
      )}
    </form>
  );
}
