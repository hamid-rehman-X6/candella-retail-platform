"use client";

import { useCallback, useEffect, useId, useState, type FormEvent } from "react";
import { Pencil, Percent, Plus } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { ConfirmDialog, Modal, modalButton } from "@/components/ui/modal";
import { toast } from "@/lib/toast";
import { formatRate } from "@/lib/tax";
import {
  createTaxRate,
  listTaxRates,
  updateTaxRate,
  type TaxRate,
} from "@/lib/tax-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

/** "18", "17.5" or "18%" to basis points; null when it is not a rate. */
function parseRate(value: string): number | null {
  const text = value.trim().replace(/%$/, "").trim();
  if (!/^\d{1,3}(\.\d{1,2})?$/.test(text)) return null;
  const bps = Math.round(Number(text) * 100);
  return bps >= 0 && bps <= 10000 ? bps : null;
}

type Form = { id?: string; name: string; rate: string; inclusive: boolean };

export default function TaxesPage() {
  const { current } = useApp();
  const manages = current.role === "owner" || current.role === "admin";
  const formId = useId();
  const [rates, setRates] = useState<TaxRate[]>([]);
  const [showArchived, setShowArchived] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [form, setForm] = useState<Form | null>(null);
  const [formError, setFormError] = useState("");
  const [busy, setBusy] = useState(false);
  const [statusTarget, setStatusTarget] = useState<TaxRate | null>(null);

  const load = useCallback(async () => {
    const data = await listTaxRates(current.id, showArchived);
    setRates(data.taxRates);
  }, [current.id, showArchived]);

  useEffect(() => {
    let cancelled = false;
    void listTaxRates(current.id, showArchived)
      .then((data) => {
        if (!cancelled) {
          setRates(data.taxRates);
          setError("");
        }
      })
      .catch((cause) => {
        if (!cancelled)
          setError(
            cause instanceof Error ? cause.message : "Could not load taxes.",
          );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, showArchived]);

  async function save(event: FormEvent) {
    event.preventDefault();
    if (!form) return;
    const rateBps = parseRate(form.rate);
    if (!form.name.trim() || rateBps === null) {
      setFormError(
        "Enter a name and a rate between 0 and 100, e.g. 18 or 17.5.",
      );
      return;
    }
    setBusy(true);
    setFormError("");
    try {
      const input = {
        name: form.name.trim(),
        rateBps,
        inclusive: form.inclusive,
      };
      if (form.id) await updateTaxRate(current.id, form.id, input);
      else await createTaxRate(current.id, input);
      toast.success(form.id ? "Tax rate updated" : "Tax rate added");
      setForm(null);
      await load();
    } catch (cause) {
      setFormError(
        cause instanceof Error ? cause.message : "Could not save the rate.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function changeStatus() {
    if (!statusTarget) return;
    const next = statusTarget.status === "active" ? "archived" : "active";
    setBusy(true);
    try {
      await updateTaxRate(current.id, statusTarget.id, { status: next });
      toast.success(
        next === "archived" ? "Tax rate archived" : "Tax rate restored",
      );
      await load();
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Could not update the rate.",
      );
    } finally {
      setBusy(false);
      setStatusTarget(null);
    }
  }

  if (!manages) {
    return (
      <p role="alert" className="text-sm text-muted">
        Only workspace owners and admins can manage taxes.
      </p>
    );
  }

  const preview = form ? parseRate(form.rate) : null;

  return (
    <div className="mx-auto max-w-4xl space-y-6 pb-10">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight">Taxes</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted">
            Set the rates you charge, then pick one on each product. Every sale
            keeps the rate it was sold at, so changing a rate here never changes
            a past receipt or report.
          </p>
        </div>
        <button
          type="button"
          onClick={() => {
            setFormError("");
            setForm({ name: "", rate: "", inclusive: true });
          }}
          className={modalButton.primary}
        >
          <Plus className="mr-1.5 h-4 w-4" /> Add tax rate
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

      <label className="inline-flex cursor-pointer items-center gap-2 text-sm text-muted">
        <input
          type="checkbox"
          checked={showArchived}
          onChange={(event) => {
            setLoading(true);
            setShowArchived(event.target.checked);
          }}
        />
        Show archived rates
      </label>

      <section className="divide-y divide-line rounded-2xl border border-line bg-surface">
        {loading ? (
          <p className="p-8 text-center text-sm text-muted">Loading taxes…</p>
        ) : rates.length === 0 ? (
          <div className="p-8 text-center text-sm text-muted">
            <Percent className="mx-auto mb-2 h-6 w-6" />
            No tax rates yet. Products are sold without tax until you add one
            and choose it on a product.
          </div>
        ) : (
          rates.map((rate) => (
            <div
              key={rate.id}
              className="flex flex-wrap items-center justify-between gap-3 p-4"
            >
              <div className="min-w-0">
                <p className="font-medium">
                  {rate.name}{" "}
                  {rate.status === "archived" && (
                    <span className="ml-1 rounded-full bg-foreground/10 px-2 py-0.5 text-xs text-muted">
                      Archived
                    </span>
                  )}
                </p>
                <p className="mt-1 text-sm text-muted">
                  {formatRate(rate.rateBps)} ·{" "}
                  {rate.inclusive
                    ? "Included in shelf prices"
                    : "Added at checkout"}{" "}
                  · {rate.productCount} product
                  {rate.productCount === 1 ? "" : "s"}
                </p>
              </div>
              <div className="flex gap-2">
                {rate.status === "active" && (
                  <button
                    type="button"
                    onClick={() => {
                      setFormError("");
                      setForm({
                        id: rate.id,
                        name: rate.name,
                        rate: String(rate.rateBps / 100),
                        inclusive: rate.inclusive,
                      });
                    }}
                    className={modalButton.secondary}
                  >
                    <Pencil className="mr-1.5 h-3.5 w-3.5" /> Edit
                  </button>
                )}
                <button
                  type="button"
                  onClick={() => setStatusTarget(rate)}
                  className={modalButton.secondary}
                >
                  {rate.status === "active" ? "Archive" : "Restore"}
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
        title={form?.id ? "Edit tax rate" : "Add tax rate"}
        description={
          form?.id
            ? "New sales use the change. Past sales keep the rate they were sold at."
            : "For example GST at 18%, included in your shelf prices."
        }
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
              {busy ? "Saving…" : form?.id ? "Save changes" : "Add tax rate"}
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
            <div className="grid gap-4 sm:grid-cols-[minmax(0,1fr)_140px]">
              <label className="block text-sm font-medium">
                Name
                <input
                  className={`${field} mt-1.5`}
                  required
                  maxLength={60}
                  placeholder="e.g. GST 18%"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </label>
              <label className="block text-sm font-medium">
                Rate (%)
                <input
                  className={`${field} mt-1.5`}
                  inputMode="decimal"
                  required
                  placeholder="18"
                  value={form.rate}
                  onChange={(e) => setForm({ ...form, rate: e.target.value })}
                />
              </label>
            </div>
            <fieldset className="space-y-2">
              <legend className="text-sm font-medium">
                How are your prices set?
              </legend>
              {[
                {
                  value: true,
                  title: "Prices already include this tax",
                  hint: "A Rs 1,180 item stays Rs 1,180; the receipt shows the tax inside it.",
                },
                {
                  value: false,
                  title: "Add this tax at checkout",
                  hint: "A Rs 1,000 item is charged Rs 1,000 plus the tax.",
                },
              ].map((option) => (
                <label
                  key={String(option.value)}
                  className={`flex cursor-pointer gap-3 rounded-xl border p-3 text-sm ${
                    form.inclusive === option.value
                      ? "border-accent bg-accent/5"
                      : "border-line"
                  }`}
                >
                  <input
                    type="radio"
                    name="inclusive"
                    className="mt-0.5"
                    checked={form.inclusive === option.value}
                    onChange={() =>
                      setForm({ ...form, inclusive: option.value })
                    }
                  />
                  <span>
                    <span className="block font-medium">{option.title}</span>
                    <span className="mt-0.5 block text-xs text-muted">
                      {option.hint}
                    </span>
                  </span>
                </label>
              ))}
            </fieldset>
            {form.rate.trim() && preview === null && (
              <p className="text-xs text-danger-ink">
                Enter the rate as a number between 0 and 100, e.g. 18 or 17.5.
              </p>
            )}
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
            ? `${statusTarget.status === "active" ? "Archive" : "Restore"} ${statusTarget.name}?`
            : ""
        }
        message={
          statusTarget?.status === "active"
            ? "It can no longer be chosen on products. A rate still used by products cannot be archived; move them first."
            : "It can be chosen on products again."
        }
        confirmLabel={
          statusTarget?.status === "active" ? "Archive rate" : "Restore rate"
        }
        tone={statusTarget?.status === "active" ? "danger" : "primary"}
        busy={busy}
        onCancel={() => setStatusTarget(null)}
        onConfirm={() => void changeStatus()}
      />
    </div>
  );
}
