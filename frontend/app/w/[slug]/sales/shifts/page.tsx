"use client";

import Link from "next/link";
import { useEffect, useState, type FormEvent } from "react";
import { ArrowLeft, Plus } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { Select } from "@/components/ui/select";
import { VarianceBadge } from "@/components/app/variance-badge";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney } from "@/lib/money";
import {
  createRegister,
  listRegisters,
  listShifts,
  type Register,
  type Shift,
} from "@/lib/register-api";
import { useStoreSelection } from "@/lib/use-store-selection";

const pageSize = 25;
const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

export default function ShiftsPage() {
  const { current } = useApp();
  const manages = current.role === "owner" || current.role === "admin";
  const {
    stores,
    storeId,
    select: selectStore,
    error: storeError,
  } = useStoreSelection(current.id);
  const [status, setStatus] = useState<"" | "open" | "closed">("");
  const [offset, setOffset] = useState(0);
  const [shifts, setShifts] = useState<Shift[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [registers, setRegisters] = useState<Register[]>([]);
  const [counterName, setCounterName] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [reload, setReload] = useState(0);

  useEffect(() => {
    if (!storeId) return;
    let cancelled = false;
    void Promise.all([
      listShifts(current.id, { storeId, status, offset, limit: pageSize }),
      listRegisters(current.id, storeId),
    ])
      .then(([shiftData, registerData]) => {
        if (cancelled) return;
        setShifts(shiftData.shifts);
        setHasMore(shiftData.hasMore);
        setRegisters(registerData.registers);
        setError("");
      })
      .catch((cause) => {
        if (cancelled) return;
        setShifts([]);
        setError(
          cause instanceof Error ? cause.message : "Could not load shifts.",
        );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, storeId, status, offset, reload]);

  async function addCounter(event: FormEvent) {
    event.preventDefault();
    if (!counterName.trim() || busy) return;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      const { register } = await createRegister(
        current.id,
        storeId,
        counterName.trim(),
      );
      setCounterName("");
      setNotice(
        `${register.name} added. Select it on the till that will use it.`,
      );
      setReload((n) => n + 1);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not add the counter.",
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-10">
      <div>
        <Link
          href={workspacePath(current.slug, "/sales")}
          className="inline-flex items-center gap-2 text-sm text-accent-ink hover:underline"
        >
          <ArrowLeft className="h-4 w-4" /> Point of sale
        </Link>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight">Shifts</h1>
        <p className="mt-1 text-sm text-muted">
          {manages
            ? "Every shift on every counter: opening cash, sales, and what was counted at closing."
            : "Your shifts: opening cash, sales, and what you counted at closing."}
        </p>
      </div>

      <div className="flex flex-wrap items-end gap-4">
        <label className="block w-full max-w-xs text-sm font-medium">
          Store
          <Select
            ariaLabel="Store"
            className={`${field} mt-2`}
            value={storeId}
            onValueChange={(id) => {
              setLoading(true);
              setOffset(0);
              selectStore(id);
            }}
            options={[
              ...(!storeId ? [{ value: "", label: "Select a store" }] : []),
              ...stores
                .filter((store) => store.accessible)
                .map((store) => ({ value: store.id, label: store.name })),
            ]}
          />
        </label>
        <label className="block w-full max-w-48 text-sm font-medium">
          Status
          <Select
            ariaLabel="Shift status"
            className={`${field} mt-2`}
            value={status}
            onValueChange={(value) => {
              setLoading(true);
              setOffset(0);
              setStatus(value as "" | "open" | "closed");
            }}
            options={[
              { value: "", label: "All shifts" },
              { value: "open", label: "Open" },
              { value: "closed", label: "Closed" },
            ]}
          />
        </label>
      </div>

      {(error || storeError) && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error || storeError}
        </p>
      )}
      {notice && (
        <p
          role="status"
          className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-success-ink"
        >
          {notice}
        </p>
      )}

      {manages && storeId && (
        <section className="rounded-2xl border border-line bg-surface p-5">
          <h2 className="text-lg font-semibold">Counters in this store</h2>
          <p className="mt-1 text-sm text-muted">
            One counter per cash drawer. Each has at most one open shift.
          </p>
          <ul className="mt-3 flex flex-wrap gap-2 text-sm">
            {registers.map((register) => (
              <li
                key={register.id}
                className="rounded-full border border-line px-3 py-1"
              >
                {register.name}
                <span className="ml-1 text-muted">
                  {register.openSession
                    ? `· open (${register.openSession.openerName})`
                    : "· closed"}
                </span>
              </li>
            ))}
          </ul>
          <form
            onSubmit={(event) => void addCounter(event)}
            className="mt-4 flex flex-wrap items-end gap-3"
          >
            <label className="min-w-48 flex-1 text-sm sm:max-w-xs">
              New counter name
              <input
                className={`${field} mt-1`}
                placeholder="Counter 2"
                maxLength={60}
                value={counterName}
                onChange={(event) => setCounterName(event.target.value)}
              />
            </label>
            <button
              type="submit"
              disabled={busy || !counterName.trim()}
              className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm font-medium hover:border-accent/50 disabled:opacity-50"
            >
              <Plus className="h-4 w-4" /> Add counter
            </button>
          </form>
        </section>
      )}

      <div className="overflow-x-auto rounded-2xl border border-line bg-surface">
        <table className="w-full min-w-205 text-left text-sm">
          <thead className="border-b border-line text-muted">
            <tr>
              <th className="p-4 font-medium">Counter</th>
              <th className="p-4 font-medium">Cashier</th>
              <th className="p-4 font-medium">Opened</th>
              <th className="p-4 font-medium">Closed</th>
              <th className="p-4 text-right font-medium">Sales</th>
              <th className="p-4 text-right font-medium">Expected cash</th>
              <th className="p-4 text-right font-medium">Counted</th>
              <th className="p-4 text-right font-medium">Difference</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line">
            {shifts.map((shift) => (
              <tr key={shift.id}>
                <td className="p-4">
                  <Link
                    href={workspacePath(
                      current.slug,
                      `/sales/shifts/${shift.id}`,
                    )}
                    className="font-medium text-accent-ink hover:underline"
                  >
                    {shift.registerName}
                  </Link>
                </td>
                <td className="p-4">{shift.openerName}</td>
                <td className="p-4">
                  {formatDateTime(shift.openedAt, current.timezone)}
                </td>
                <td className="p-4">
                  {shift.closedAt ? (
                    formatDateTime(shift.closedAt, current.timezone)
                  ) : (
                    <span className="rounded-full bg-emerald-500/15 px-2 py-0.5 text-xs font-medium text-success-ink">
                      Open
                    </span>
                  )}
                </td>
                <td className="p-4 text-right">{shift.salesCount}</td>
                <td className="p-4 text-right">
                  {formatMoney(shift.expectedCashMinor, shift.currencyCode)}
                </td>
                <td className="p-4 text-right">
                  {shift.countedCashMinor === null
                    ? "—"
                    : formatMoney(shift.countedCashMinor, shift.currencyCode)}
                </td>
                <td className="p-4 text-right">
                  {shift.varianceMinor === null ? (
                    "—"
                  ) : (
                    <VarianceBadge
                      varianceMinor={shift.varianceMinor}
                      currencyCode={shift.currencyCode}
                    />
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!loading && !error && shifts.length === 0 && (
          <p className="p-8 text-center text-sm text-muted">
            No shifts yet. Open one from the point of sale.
          </p>
        )}
        {loading && (
          <p className="p-8 text-center text-sm text-muted">Loading shifts…</p>
        )}
      </div>
      {(offset > 0 || hasMore) && (
        <div className="flex justify-end gap-3 text-sm">
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
      )}
    </div>
  );
}
