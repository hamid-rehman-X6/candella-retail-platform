"use client";

import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { AlertTriangle, X } from "lucide-react";
import { workspacePath } from "@/components/app/app-context";
import { Select } from "@/components/ui/select";
import {
  listLowStock,
  saveReorder,
  type ReorderItem,
  type StockItem,
} from "@/lib/catalog-api";
import { listSuppliers, type Supplier } from "@/lib/suppliers-api";
import { toast } from "@/lib/toast";

type Props = {
  tenantId: string;
  slug: string;
  storeId: string;
  version: number;
  selected: StockItem | null;
  onClose: () => void;
  onConfigured: () => void;
};

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";

export function ReorderPanel({
  tenantId,
  slug,
  storeId,
  version,
  selected,
  onClose,
  onConfigured,
}: Props) {
  const [alerts, setAlerts] = useState<ReorderItem[]>([]);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<StockItem | null>(null);
  const [threshold, setThreshold] = useState("");
  const [supplierId, setSupplierId] = useState("");
  const [supplierSearch, setSupplierSearch] = useState("");
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [busy, setBusy] = useState(false);
  const request = useRef(0);

  useEffect(() => {
    if (!storeId) return;
    let cancelled = false;
    const id = ++request.current;
    const timer = setTimeout(() => {
      setLoading(true);
      void listLowStock(tenantId, storeId)
        .then(({ items }) => {
          if (cancelled || id !== request.current) return;
          setAlerts(items);
          setOffset(0);
          setHasMore(items.length === 100);
          setError("");
        })
        .catch((e) => {
          if (!cancelled && id === request.current)
            setError(
              e instanceof Error ? e.message : "Could not load stock alerts.",
            );
        })
        .finally(() => {
          if (!cancelled && id === request.current) setLoading(false);
        });
    }, 0);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [tenantId, storeId, version]);

  useEffect(() => {
    if (!selected) return;
    const timer = setTimeout(() => {
      setEditing(selected);
      setThreshold(
        selected.reorderThreshold == null
          ? ""
          : String(selected.reorderThreshold),
      );
      setSupplierId(selected.configuredSupplierId ?? "");
      setSupplierSearch("");
    }, 0);
    return () => clearTimeout(timer);
  }, [selected]);

  useEffect(() => {
    if (!editing) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      void listSuppliers(tenantId, supplierSearch)
        .then(({ suppliers }) => {
          if (!cancelled)
            setSuppliers(
              suppliers.filter((supplier) => supplier.status === "active"),
            );
        })
        .catch((e) => {
          if (!cancelled)
            setError(
              e instanceof Error ? e.message : "Could not load suppliers.",
            );
        });
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [tenantId, editing, supplierSearch]);

  function close() {
    if (!busy) {
      setEditing(null);
      onClose();
    }
  }

  async function loadMore() {
    if (!storeId || loading) return;
    const id = request.current;
    setLoading(true);
    try {
      const next = offset + 100;
      const result = await listLowStock(tenantId, storeId, next);
      if (id !== request.current) return;
      setAlerts((old) => [...old, ...result.items]);
      setOffset(next);
      setHasMore(result.items.length === 100);
    } catch (e) {
      if (id === request.current)
        setError(
          e instanceof Error ? e.message : "Could not load more alerts.",
        );
    } finally {
      if (id === request.current) setLoading(false);
    }
  }

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!editing || busy) return;
    const parsed = threshold.trim() === "" ? null : Number(threshold);
    if (
      parsed !== null &&
      (!Number.isSafeInteger(parsed) || parsed < 0 || parsed > 1_000_000_000)
    ) {
      setError("Enter a whole-number threshold between 0 and 1,000,000,000.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await saveReorder(tenantId, editing.variantId, {
        storeId,
        threshold: parsed,
        preferredSupplierId: parsed === null ? null : supplierId || null,
      });
      toast.success(
        parsed === null ? "Stock alert turned off" : "Stock alert saved",
      );
      setEditing(null);
      onClose();
      onConfigured();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save stock alert.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <section className="rounded-2xl border border-line bg-surface p-5">
        <div className="flex items-center gap-2">
          <AlertTriangle
            className="h-5 w-5 text-warning-ink"
            aria-hidden="true"
          />
          <h2 className="text-lg font-semibold">Low stock</h2>
          {!loading && (
            <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-warning-ink">
              {alerts.length}
              {hasMore ? "+" : ""}
            </span>
          )}
        </div>
        <p className="mt-1 text-sm text-muted">
          Items at or below their alert level in this store. Set an alert on any
          product below.
        </p>
        {error && !editing && (
          <p role="alert" className="mt-3 text-sm text-danger-ink">
            {error}
          </p>
        )}
        {loading && alerts.length === 0 ? (
          <p className="mt-4 text-sm text-muted">Loading alerts…</p>
        ) : alerts.length === 0 ? (
          <p className="mt-4 text-sm text-muted">
            No low-stock alerts for this store.
          </p>
        ) : (
          <div className="mt-4 space-y-2">
            {alerts.map((item) => {
              const params = new URLSearchParams({ sku: item.sku });
              if (item.preferredSupplierId)
                params.set("supplierId", item.preferredSupplierId);
              if (item.preferredSupplierName)
                params.set("supplierName", item.preferredSupplierName);
              return (
                <div
                  key={item.variantId}
                  className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line p-3 text-sm"
                >
                  <div>
                    <p className="font-medium">
                      {item.productName} · {item.variantName}
                    </p>
                    <p className="text-muted">
                      {item.sku} · {item.quantity} on hand · alert at{" "}
                      {item.reorderThreshold} · suggested{" "}
                      {item.suggestedQuantity} to move above alert level
                    </p>
                    {item.preferredSupplierName && (
                      <p className="text-muted">
                        Supplier: {item.preferredSupplierName}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        setEditing(item);
                        setThreshold(String(item.reorderThreshold));
                        setSupplierId(item.configuredSupplierId ?? "");
                        setSupplierSearch("");
                      }}
                      className="cursor-pointer rounded-lg border border-line px-3 py-2 hover:border-accent"
                    >
                      Edit alert
                    </button>
                    <Link
                      href={`${workspacePath(slug, "/purchases")}?${params}`}
                      className="rounded-lg bg-accent px-3 py-2 font-medium text-black hover:bg-accent-bright"
                    >
                      Create purchase order
                    </Link>
                  </div>
                </div>
              );
            })}
          </div>
        )}
        {hasMore && (
          <button
            type="button"
            onClick={() => void loadMore()}
            disabled={loading}
            className="mt-3 cursor-pointer rounded-lg border border-line px-3 py-2 text-sm disabled:opacity-50"
          >
            Load more alerts
          </button>
        )}
      </section>
      {editing && (
        <div
          className="fixed inset-0 z-[70] flex items-center justify-center bg-black/75 p-4"
          role="presentation"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) close();
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Stock alert settings"
            className="w-full max-w-md rounded-2xl border border-line bg-canvas-raised p-6 shadow-2xl"
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 className="text-xl font-semibold">Stock alert</h2>
                <p className="mt-1 text-sm text-muted">
                  {editing.productName} · {editing.variantName}
                </p>
              </div>
              <button
                type="button"
                onClick={close}
                disabled={busy}
                aria-label="Close"
                className="cursor-pointer text-muted"
              >
                <X size={20} />
              </button>
            </div>
            <form onSubmit={(e) => void save(e)} className="mt-5 space-y-4">
              <label className="block text-sm">
                Alert when stock is at or below
                <input
                  className={`${field} mt-1`}
                  type="number"
                  min={0}
                  max={1000000000}
                  step={1}
                  value={threshold}
                  onChange={(e) => setThreshold(e.target.value)}
                  placeholder="Leave blank to turn off"
                />
              </label>
              <div>
                <label htmlFor="supplier-search" className="text-sm">
                  Preferred supplier (optional)
                </label>
                <input
                  id="supplier-search"
                  className={`${field} mt-1`}
                  value={supplierSearch}
                  onChange={(e) => setSupplierSearch(e.target.value)}
                  placeholder="Search suppliers"
                />
                <Select
                  ariaLabel="Preferred supplier"
                  className={`${field} mt-2`}
                  value={supplierId}
                  onValueChange={setSupplierId}
                  options={[
                    { value: "", label: "Use linked supplier when available" },
                    ...(!suppliers.some((s) => s.id === supplierId) &&
                    supplierId &&
                    editing.preferredSupplierName
                      ? [
                          {
                            value: supplierId,
                            label: editing.preferredSupplierName,
                          },
                        ]
                      : []),
                    ...suppliers.map((s) => ({ value: s.id, label: s.name })),
                  ]}
                />
              </div>
              <p className="text-xs text-muted">
                Stock changes update this alert automatically. Leaving the
                threshold blank turns it off.
              </p>
              {error && (
                <p role="alert" className="text-sm text-danger-ink">
                  {error}
                </p>
              )}
              <button
                type="submit"
                disabled={busy}
                className="w-full cursor-pointer rounded-xl bg-accent px-4 py-2.5 font-semibold text-black disabled:opacity-50"
              >
                {busy ? "Saving…" : "Save alert"}
              </button>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
