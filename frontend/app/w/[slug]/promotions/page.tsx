"use client";

import { useCallback, useEffect, useId, useState, type FormEvent } from "react";
import { Pencil, Plus, Tag } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { ConfirmDialog, Modal, modalButton } from "@/components/ui/modal";
import { Select } from "@/components/ui/select";
import { formatMoney, minorDigits, parseMoney } from "@/lib/money";
import { toast } from "@/lib/toast";
import { formatRate } from "@/lib/tax";
import {
  createPromotion,
  getDiscountSettings,
  listPromotions,
  saveDiscountSettings,
  updatePromotion,
  type Promotion,
  type PromotionInput,
} from "@/lib/promotions-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

/** "10", "12.5" or "10%" to basis points; null when not a percentage. */
function parsePercent(value: string): number | null {
  const text = value.trim().replace(/%$/, "").trim();
  if (!/^\d{1,3}(\.\d{1,2})?$/.test(text)) return null;
  const bps = Math.round(Number(text) * 100);
  return bps >= 0 && bps <= 10000 ? bps : null;
}

type Form = {
  id?: string;
  name: string;
  kind: "percent" | "amount";
  value: string;
  appliesTo: "order" | "item";
  minimum: string;
  startsOn: string;
  endsOn: string;
};

const emptyForm: Form = {
  name: "",
  kind: "percent",
  value: "",
  appliesTo: "order",
  minimum: "",
  startsOn: "",
  endsOn: "",
};

export default function PromotionsPage() {
  const { current } = useApp();
  const manages = current.role === "owner" || current.role === "admin";
  const formId = useId();
  const [promotions, setPromotions] = useState<Promotion[]>([]);
  const [showArchived, setShowArchived] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [form, setForm] = useState<Form | null>(null);
  const [formError, setFormError] = useState("");
  const [busy, setBusy] = useState(false);
  const [statusTarget, setStatusTarget] = useState<Promotion | null>(null);
  const [limit, setLimit] = useState("");
  const [savedLimit, setSavedLimit] = useState<number | null>(null);

  const load = useCallback(async () => {
    const data = await listPromotions(current.id, showArchived);
    setPromotions(data.promotions);
  }, [current.id, showArchived]);

  useEffect(() => {
    let cancelled = false;
    void listPromotions(current.id, showArchived)
      .then((data) => {
        if (!cancelled) {
          setPromotions(data.promotions);
          setError("");
        }
      })
      .catch((cause) => {
        if (!cancelled)
          setError(
            cause instanceof Error
              ? cause.message
              : "Could not load promotions.",
          );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, showArchived]);

  useEffect(() => {
    let cancelled = false;
    void getDiscountSettings(current.id)
      .then(({ settings }) => {
        if (cancelled) return;
        setSavedLimit(settings.cashierLimitBps);
        setLimit(String(settings.cashierLimitBps / 100));
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, [current.id]);

  function describe(p: Promotion) {
    const amount =
      p.kind === "percent"
        ? `${formatRate(p.percentBps ?? 0)} off`
        : `${formatMoney(p.amountMinor ?? 0, current.currencyCode)} off`;
    const scope = p.appliesTo === "order" ? "the bill" : "each item unit";
    const parts = [`${amount} ${scope}`];
    if (p.minSubtotalMinor > 0)
      parts.push(
        `bills of ${formatMoney(p.minSubtotalMinor, current.currencyCode)}+`,
      );
    if (p.startsOn || p.endsOn)
      parts.push(`${p.startsOn ?? "now"} to ${p.endsOn ?? "no end"}`);
    return parts.join(" · ");
  }

  function edit(p: Promotion) {
    const scale = 10 ** minorDigits(current.currencyCode);
    setFormError("");
    setForm({
      id: p.id,
      name: p.name,
      kind: p.kind,
      value:
        p.kind === "percent"
          ? String((p.percentBps ?? 0) / 100)
          : String((p.amountMinor ?? 0) / scale),
      appliesTo: p.appliesTo,
      minimum: p.minSubtotalMinor ? String(p.minSubtotalMinor / scale) : "",
      startsOn: p.startsOn ?? "",
      endsOn: p.endsOn ?? "",
    });
  }

  function toInput(f: Form): PromotionInput | string {
    const percentBps = f.kind === "percent" ? parsePercent(f.value) : null;
    const amountMinor =
      f.kind === "amount" ? parseMoney(f.value, current.currencyCode) : null;
    const minimum = f.minimum.trim()
      ? parseMoney(f.minimum, current.currencyCode)
      : 0;
    if (!f.name.trim())
      return "Enter a name customers and staff will recognise.";
    if (f.kind === "percent" && (percentBps === null || percentBps < 1))
      return "Enter a percentage between 0.01 and 100.";
    if (f.kind === "amount" && (amountMinor === null || amountMinor < 1))
      return "Enter an amount above zero.";
    if (minimum === null) return "Enter the minimum bill as an amount.";
    if (f.startsOn && f.endsOn && f.endsOn < f.startsOn)
      return "The end date must be on or after the start date.";
    return {
      name: f.name.trim(),
      kind: f.kind,
      percentBps,
      amountMinor,
      appliesTo: f.appliesTo,
      minSubtotalMinor: f.appliesTo === "order" ? minimum : 0,
      startsOn: f.startsOn || null,
      endsOn: f.endsOn || null,
    };
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (!form) return;
    const input = toInput(form);
    if (typeof input === "string") {
      setFormError(input);
      return;
    }
    setBusy(true);
    setFormError("");
    try {
      if (form.id) await updatePromotion(current.id, form.id, input);
      else await createPromotion(current.id, input);
      toast.success(form.id ? "Promotion updated" : "Promotion added");
      setForm(null);
      await load();
    } catch (cause) {
      setFormError(
        cause instanceof Error
          ? cause.message
          : "Could not save the promotion.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function changeStatus() {
    const p = statusTarget;
    if (!p) return;
    setBusy(true);
    try {
      await updatePromotion(current.id, p.id, {
        name: p.name,
        kind: p.kind,
        percentBps: p.percentBps,
        amountMinor: p.amountMinor,
        appliesTo: p.appliesTo,
        minSubtotalMinor: p.minSubtotalMinor,
        startsOn: p.startsOn,
        endsOn: p.endsOn,
        status: p.status === "active" ? "archived" : "active",
      });
      toast.success(
        p.status === "active" ? "Promotion ended" : "Promotion restored",
      );
      await load();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Could not update the promotion.",
      );
    } finally {
      setBusy(false);
      setStatusTarget(null);
    }
  }

  async function saveLimit(event: FormEvent) {
    event.preventDefault();
    const bps = parsePercent(limit);
    if (bps === null) {
      setError("Enter the cashier limit as a percentage between 0 and 100.");
      return;
    }
    setError("");
    try {
      const { settings } = await saveDiscountSettings(current.id, bps);
      setSavedLimit(settings.cashierLimitBps);
      toast.success("Cashier discount limit saved");
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not save the limit.",
      );
    }
  }

  if (!manages) {
    return (
      <p role="alert" className="text-sm text-muted">
        Only workspace owners and admins can manage promotions.
      </p>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-6 pb-10">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Promotions</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted">
            Approved discounts cashiers can apply at the till. Prices stay the
            same; each sale keeps the discount it got, so editing or ending a
            promotion never changes a past receipt.
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            setFormError("");
            setForm(emptyForm);
          }}
          className={modalButton.primary}
        >
          <Plus className="mr-1.5 h-4 w-4" /> Add promotion
        </button>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}

      <form
        onSubmit={(event) => void saveLimit(event)}
        className="flex flex-wrap items-end gap-3 rounded-2xl border border-line bg-surface p-5"
      >
        <label className="min-w-48 flex-1 text-sm font-medium">
          Manual discount cashiers may give (%)
          <input
            className={`${field} mt-1.5 max-w-40`}
            inputMode="decimal"
            value={limit}
            onChange={(event) => setLimit(event.target.value)}
          />
          <span className="mt-1 block text-xs font-normal text-muted">
            0 means only owners and admins can give a manual discount. Every
            manual discount needs a reason and is kept on the receipt.
            {savedLimit !== null && ` Now: ${formatRate(savedLimit)}.`}
          </span>
        </label>
        <button type="submit" className={modalButton.secondary}>
          Save limit
        </button>
      </form>

      <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-muted">
        <input
          type="checkbox"
          checked={showArchived}
          onChange={(event) => {
            setLoading(true);
            setShowArchived(event.target.checked);
          }}
        />
        Show ended promotions
      </label>

      <section className="divide-y divide-line rounded-2xl border border-line bg-surface">
        {loading ? (
          <p className="p-8 text-center text-sm text-muted">
            Loading promotions…
          </p>
        ) : promotions.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted">
            <Tag className="mx-auto mb-2 h-6 w-6" />
            No promotions yet. Add one, such as 10% off for Eid.
          </div>
        ) : (
          promotions.map((p) => (
            <div
              key={p.id}
              className="flex flex-wrap items-center justify-between gap-3 p-4"
            >
              <div className="min-w-0">
                <p className="font-medium">
                  {p.name}
                  {p.status === "archived" && (
                    <span className="ml-2 rounded-full bg-foreground/10 px-2 py-0.5 text-xs text-muted">
                      Ended
                    </span>
                  )}
                </p>
                <p className="mt-1 text-sm text-muted">{describe(p)}</p>
              </div>
              <div className="flex gap-2">
                {p.status === "active" && (
                  <button
                    type="button"
                    onClick={() => edit(p)}
                    className={modalButton.secondary}
                  >
                    <Pencil className="mr-1.5 h-3.5 w-3.5" /> Edit
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setStatusTarget(p)}
                  className={modalButton.secondary}
                >
                  {p.status === "active" ? "End" : "Restore"}
                </button>
              </div>
            </div>
          ))
        )}
      </section>

      <Modal
        open={!!form}
        onClose={() => setForm(null)}
        busy={busy}
        size="lg"
        title={form?.id ? "Edit promotion" : "Add promotion"}
        description="Cashiers can apply it at the till while it is running."
        footer={
          <>
            <button
              type="button"
              disabled={busy}
              onClick={() => setForm(null)}
              className={modalButton.secondary}
            >
              Cancel
            </button>
            <button
              type="submit"
              form={formId}
              disabled={busy}
              className={modalButton.primary}
            >
              {busy ? "Saving…" : form?.id ? "Save changes" : "Add promotion"}
            </button>
          </>
        }
      >
        {form && (
          <form
            id={formId}
            onSubmit={(e) => void save(e)}
            className="space-y-4"
          >
            <label className="block text-sm font-medium">
              Name
              <input
                className={`${field} mt-1.5`}
                required
                maxLength={80}
                placeholder="e.g. Eid 10% off"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </label>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <p className="text-sm font-medium">Applies to</p>
                <Select
                  ariaLabel="Applies to"
                  className={`${field} mt-1.5`}
                  value={form.appliesTo}
                  onValueChange={(value) =>
                    setForm({ ...form, appliesTo: value as Form["appliesTo"] })
                  }
                  options={[
                    { value: "order", label: "The whole bill" },
                    { value: "item", label: "Single items (per unit)" },
                  ]}
                />
              </div>
              <div>
                <p className="text-sm font-medium">Discount type</p>
                <Select
                  ariaLabel="Discount type"
                  className={`${field} mt-1.5`}
                  value={form.kind}
                  onValueChange={(value) =>
                    setForm({ ...form, kind: value as Form["kind"] })
                  }
                  options={[
                    { value: "percent", label: "Percentage off" },
                    {
                      value: "amount",
                      label: `Amount off (${current.currencyCode})`,
                    },
                  ]}
                />
              </div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block text-sm font-medium">
                {form.kind === "percent"
                  ? "Percentage off"
                  : form.appliesTo === "item"
                    ? `Amount off each unit (${current.currencyCode})`
                    : `Amount off the bill (${current.currencyCode})`}
                <input
                  className={`${field} mt-1.5`}
                  inputMode="decimal"
                  required
                  placeholder={form.kind === "percent" ? "10" : "200"}
                  value={form.value}
                  onChange={(e) => setForm({ ...form, value: e.target.value })}
                />
              </label>
              {form.appliesTo === "order" && (
                <label className="block text-sm font-medium">
                  Minimum bill{" "}
                  <span className="font-normal text-muted">(optional)</span>
                  <input
                    className={`${field} mt-1.5`}
                    inputMode="decimal"
                    placeholder="0"
                    value={form.minimum}
                    onChange={(e) =>
                      setForm({ ...form, minimum: e.target.value })
                    }
                  />
                </label>
              )}
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <label className="block text-sm font-medium">
                Starts{" "}
                <span className="font-normal text-muted">(optional)</span>
                <input
                  type="date"
                  className={`${field} mt-1.5`}
                  value={form.startsOn}
                  onChange={(e) =>
                    setForm({ ...form, startsOn: e.target.value })
                  }
                />
              </label>
              <label className="block text-sm font-medium">
                Ends <span className="font-normal text-muted">(optional)</span>
                <input
                  type="date"
                  className={`${field} mt-1.5`}
                  value={form.endsOn}
                  onChange={(e) => setForm({ ...form, endsOn: e.target.value })}
                />
              </label>
            </div>
            {formError && (
              <p
                role="alert"
                className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
              >
                {formError}
              </p>
            )}
          </form>
        )}
      </Modal>

      <ConfirmDialog
        open={!!statusTarget}
        title={
          statusTarget
            ? `${statusTarget.status === "active" ? "End" : "Restore"} ${statusTarget.name}?`
            : ""
        }
        message={
          statusTarget?.status === "active"
            ? "Cashiers can no longer apply it. Sales that already used it keep their discount."
            : "Cashiers can apply it again while its dates allow."
        }
        confirmLabel={
          statusTarget?.status === "active" ? "End promotion" : "Restore"
        }
        tone={statusTarget?.status === "active" ? "danger" : "primary"}
        busy={busy}
        onCancel={() => setStatusTarget(null)}
        onConfirm={() => void changeStatus()}
      />
    </div>
  );
}
