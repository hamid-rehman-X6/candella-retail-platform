"use client";

import Link from "next/link";
import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ArrowDownToLine, ClipboardCheck, Search, X } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import {
  getCatalogSummary,
  listMovements,
  listStock,
  recordMovement,
  type Movement,
  type CatalogSummary,
  type StockItem,
} from "@/lib/catalog-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:opacity-50";

export default function InventoryPage() {
  const { current } = useApp();
  const editable = current.role === "owner" || current.role === "admin";
  const [items, setItems] = useState<StockItem[]>([]);
  const [movements, setMovements] = useState<Movement[]>([]);
  const [summary, setSummary] = useState<CatalogSummary | null>(null);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<StockItem | null>(null);
  const [kind, setKind] = useState<"receive" | "count">("receive");
  const [quantity, setQuantity] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  const refresh = useCallback(
    async (q = search) => {
      setLoading(true);
      setError("");
      try {
        const [stock, ledger, counts] = await Promise.all([
          listStock(current.id, q),
          listMovements(current.id),
          getCatalogSummary(current.id),
        ]);
        setItems(stock.items);
        setMovements(ledger.movements);
        setSummary(counts.summary);
        setOffset(0);
        setHasMore(stock.items.length === 100);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Could not load inventory.");
      } finally {
        setLoading(false);
      }
    },
    [current.id, search],
  );

  useEffect(() => {
    const timer = setTimeout(() => {
      void refresh(search);
    }, 250);
    return () => clearTimeout(timer);
  }, [refresh, search]);

  function open(item: StockItem, nextKind: "receive" | "count") {
    setSelected(item);
    setKind(nextKind);
    setQuantity(nextKind === "count" ? String(item.quantity) : "");
    setNote("");
    setError("");
  }
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!selected) return;
    const amount = Number(quantity);
    if (
      !Number.isSafeInteger(amount) ||
      amount < 0 ||
      amount > 1_000_000_000 ||
      (kind === "receive" && amount === 0)
    ) {
      setError(
        kind === "receive"
          ? "Enter a whole number greater than zero."
          : "Enter a whole number of zero or more.",
      );
      return;
    }
    if (kind === "count" && !note.trim()) {
      setError("Add a reason for this count correction.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (kind === "receive")
        await recordMovement(current.id, {
          variantId: selected.variantId,
          kind,
          quantity: amount,
          note,
        });
      else
        await recordMovement(current.id, {
          variantId: selected.variantId,
          kind,
          countedQuantity: amount,
          note,
        });
      setSelected(null);
      await refresh();
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "Could not save stock movement.",
      );
    } finally {
      setBusy(false);
    }
  }
  async function loadMore() {
    const nextOffset = offset + 100;
    try {
      const result = await listStock(current.id, search, nextOffset);
      setItems((previous) => [...previous, ...result.items]);
      setOffset(nextOffset);
      setHasMore(result.items.length === 100);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not load more stock.");
    }
  }

  return (
    <div className="space-y-7">
      <div>
        <h1 className="text-2xl font-semibold">Inventory</h1>
        <p className="mt-1 text-sm text-muted">
          Stock at Main store. Every change is recorded in the movement history.
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        {[
          ["Active variants", summary?.variantCount ?? "—"],
          ["Units on hand", summary?.unitsOnHand ?? "—"],
          ["Out of stock", summary?.outOfStock ?? "—"],
        ].map(([label, value]) => (
          <div
            key={label}
            className="rounded-2xl border border-line bg-surface p-5"
          >
            <p className="text-sm text-muted">{label}</p>
            <p className="mt-2 text-2xl font-semibold">{value}</p>
          </div>
        ))}
      </div>
      <label className="relative block max-w-xl">
        <span className="sr-only">Search inventory</span>
        <Search className="absolute top-3 left-3 text-muted" size={17} />
        <input
          className={`${field} pl-10`}
          placeholder="Search product, variant, SKU or barcode"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </label>
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}
      {loading ? (
        <p className="text-sm text-muted">Loading inventory…</p>
      ) : items.length === 0 ? (
        <div className="rounded-2xl border border-line bg-surface p-10 text-center">
          <h2 className="font-semibold">
            {search ? "No matching stock" : "No products to stock yet"}
          </h2>
          <p className="mt-1 text-sm text-muted">
            {search
              ? "Try a different search."
              : "Create a product first, then receive its starting stock."}
          </p>
          {!search && (
            <Link
              className="mt-4 inline-block text-sm text-accent"
              href={workspacePath(current.slug, "/products")}
            >
              Go to Products →
            </Link>
          )}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-2xl border border-line bg-surface">
          <table className="w-full min-w-[650px] text-left text-sm">
            <thead className="border-b border-line text-muted">
              <tr>
                <th className="px-5 py-3 font-medium">Product / variant</th>
                <th className="px-5 py-3 font-medium">SKU</th>
                <th className="px-5 py-3 font-medium">On hand</th>
                <th className="px-5 py-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {items.map((item) => (
                <tr key={item.variantId}>
                  <td className="px-5 py-4">
                    <span className="font-medium">{item.productName}</span>
                    <span className="ml-2 text-muted">{item.variantName}</span>
                  </td>
                  <td className="px-5 py-4 font-mono text-muted">{item.sku}</td>
                  <td
                    className={`px-5 py-4 font-semibold ${item.quantity === 0 ? "text-amber-400" : ""}`}
                  >
                    {item.quantity}
                  </td>
                  <td className="px-5 py-4">
                    {editable && (
                      <div className="flex gap-2">
                        <button
                          type="button"
                          onClick={() => open(item, "receive")}
                          className="rounded-lg border border-line px-2 py-1.5 text-xs hover:border-accent"
                        >
                          Receive
                        </button>
                        <button
                          type="button"
                          onClick={() => open(item, "count")}
                          className="rounded-lg border border-line px-2 py-1.5 text-xs hover:border-accent"
                        >
                          Correct count
                        </button>
                      </div>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {hasMore && (
        <button
          type="button"
          onClick={() => void loadMore()}
          className="rounded-xl border border-line px-4 py-2 text-sm"
        >
          Load more
        </button>
      )}
      <section>
        <h2 className="text-lg font-semibold">Recent movements</h2>
        <p className="mt-1 text-sm text-muted">
          Receipts and count corrections appear here after they are saved.
        </p>
        <div className="mt-4 space-y-2">
          {movements.length === 0 ? (
            <p className="rounded-xl border border-line p-5 text-sm text-muted">
              No stock movements yet.
            </p>
          ) : (
            movements.map((m) => (
              <div
                key={m.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-line bg-surface p-4 text-sm"
              >
                <div>
                  <p className="font-medium">
                    {m.productName} · {m.variantName}
                  </p>
                  <p className="mt-1 text-muted">
                    {m.kind === "receive" ? "Received" : "Count corrected"} ·{" "}
                    {m.sku} · {new Date(m.createdAt).toLocaleString()}
                  </p>
                  {m.note && <p className="mt-1 text-muted">{m.note}</p>}
                </div>
                <div className="text-right">
                  <p
                    className={
                      m.delta >= 0 ? "text-emerald-400" : "text-amber-400"
                    }
                  >
                    {m.delta >= 0 ? "+" : ""}
                    {m.delta}
                  </p>
                  <p className="text-xs text-muted">
                    {m.quantityBefore} → {m.quantityAfter}
                  </p>
                </div>
              </div>
            ))
          )}
        </div>
      </section>
      {selected && (
        <div
          className="fixed inset-0 z-[70] flex items-center justify-center bg-black/75 p-4"
          role="presentation"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setSelected(null);
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={
              kind === "receive" ? "Receive stock" : "Correct stock count"
            }
            className="w-full max-w-md rounded-2xl border border-line bg-canvas-raised p-6 shadow-2xl"
          >
            <div className="flex justify-between">
              <div className="flex items-center gap-2">
                {kind === "receive" ? (
                  <ArrowDownToLine className="text-accent" />
                ) : (
                  <ClipboardCheck className="text-accent" />
                )}
                <h2 className="text-xl font-semibold">
                  {kind === "receive" ? "Receive stock" : "Correct count"}
                </h2>
              </div>
              <button
                type="button"
                onClick={() => setSelected(null)}
                aria-label="Close"
                className="text-muted"
              >
                <X size={20} />
              </button>
            </div>
            <p className="mt-3 text-sm text-muted">
              {selected.productName} · {selected.variantName} · Currently{" "}
              {selected.quantity} on hand
            </p>
            <form onSubmit={(e) => void submit(e)} className="mt-5 space-y-4">
              <label className="block text-sm">
                {kind === "receive" ? "Quantity received" : "Counted quantity"}
                <input
                  className={`${field} mt-1`}
                  type="number"
                  min={kind === "receive" ? 1 : 0}
                  step={1}
                  required
                  value={quantity}
                  onChange={(e) => setQuantity(e.target.value)}
                />
              </label>
              <label className="block text-sm">
                Reason / reference{" "}
                {kind === "receive" ? "(optional)" : "(required)"}
                <textarea
                  className={`${field} mt-1`}
                  rows={2}
                  maxLength={500}
                  required={kind === "count"}
                  placeholder={
                    kind === "receive"
                      ? "e.g. Supplier delivery 1024"
                      : "e.g. Physical count after damage"
                  }
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
              </label>
              {error && (
                <p role="alert" className="text-sm text-red-300">
                  {error}
                </p>
              )}
              <button
                type="submit"
                disabled={busy}
                className={`${button} w-full`}
              >
                {busy ? "Saving…" : "Save movement"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
