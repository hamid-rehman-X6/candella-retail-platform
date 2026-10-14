"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Minus, Plus, Search, ShoppingCart, Trash2 } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { useStoreSelection } from "@/lib/use-store-selection";
import {
  formatMoney,
  listStock,
  minorDigits,
  parseMoney,
  type StockItem,
} from "@/lib/catalog-api";
import { checkout } from "@/lib/sales-api";

type CartLine = { item: StockItem; quantity: number };

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50";

export default function SalesPage() {
  const { current } = useApp();
  const {
    stores,
    storeId,
    selectedStore,
    select: selectStore,
    error: storeError,
  } = useStoreSelection(current.id);
  const router = useRouter();
  const searchRef = useRef<HTMLInputElement>(null);
  const requestKey = useRef<string | null>(null);
  const scanGeneration = useRef(0);
  const [search, setSearch] = useState("");
  const [results, setResults] = useState<StockItem[]>([]);
  const [cart, setCart] = useState<CartLine[]>([]);
  const [cash, setCash] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  function changeStore(id: string) {
    if (busy) return;
    scanGeneration.current++;
    setCart([]);
    setResults([]);
    setCash("");
    setSearch("");
    requestKey.current = null;
    setLoading(true);
    selectStore(id);
  }

  useEffect(() => {
    if (!storeId) return;
    let cancelled = false;
    const generation = scanGeneration.current;
    const timer = setTimeout(
      async () => {
        try {
          const data = await listStock(current.id, storeId, search);
          if (!cancelled && generation === scanGeneration.current) {
            setResults(data.items);
            setError("");
          }
        } catch (e) {
          if (!cancelled && generation === scanGeneration.current) {
            setResults([]);
            setError(
              e instanceof Error ? e.message : "Could not load products.",
            );
          }
        } finally {
          if (!cancelled && generation === scanGeneration.current)
            setLoading(false);
        }
      },
      search ? 200 : 0,
    );
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, storeId, search]);

  function add(item: StockItem) {
    if (item.quantity <= 0) {
      setError(`${item.sku} is out of stock. Receive stock before selling it.`);
      return;
    }
    const existing = cart.find(
      (line) => line.item.variantId === item.variantId,
    );
    if (existing && existing.quantity >= item.quantity) {
      setError(`Only ${item.quantity} unit(s) of ${item.sku} are available.`);
      return;
    }
    setCart((old) => {
      const currentLine = old.find(
        (line) => line.item.variantId === item.variantId,
      );
      if (currentLine) {
        return old.map((line) =>
          line.item.variantId === item.variantId
            ? { item, quantity: Math.min(line.quantity + 1, item.quantity) }
            : line,
        );
      }
      return [...old, { item, quantity: 1 }];
    });
    requestKey.current = null;
    setError("");
    setSearch("");
    searchRef.current?.focus();
  }

  async function scan(e: FormEvent) {
    e.preventDefault();
    if (!storeId) return;
    const query = search.trim();
    if (!query) return;
    const generation = scanGeneration.current;
    try {
      // Fetch on Enter as well, so a fast barcode scanner does not wait for debounce.
      const data = await listStock(current.id, storeId, query);
      if (generation !== scanGeneration.current) return;
      setResults(data.items);
      const exact = data.items.find(
        (item) =>
          item.sku.toLowerCase() === query.toLowerCase() ||
          item.barcode?.toLowerCase() === query.toLowerCase(),
      );
      if (exact) add(exact);
      else if (data.items.length === 0)
        setError("No matching product. Check the SKU or barcode.");
    } catch (e) {
      if (generation !== scanGeneration.current) return;
      setError(e instanceof Error ? e.message : "Could not search products.");
    }
  }

  function changeQuantity(variantId: string, quantity: number) {
    if (!Number.isSafeInteger(quantity) || quantity < 1) return;
    setCart((old) =>
      old.map((line) =>
        line.item.variantId === variantId
          ? { ...line, quantity: Math.min(quantity, line.item.quantity) }
          : line,
      ),
    );
    requestKey.current = null;
    setError("");
  }

  function remove(variantId: string) {
    setCart((old) => old.filter((line) => line.item.variantId !== variantId));
    requestKey.current = null;
    setError("");
  }

  const total = cart.reduce(
    (sum, line) => sum + line.item.priceMinor * line.quantity,
    0,
  );
  const cashMinor = parseMoney(cash, current.currencyCode);
  const change = cashMinor === null ? null : cashMinor - total;
  const canSell =
    !!storeId &&
    cart.length > 0 &&
    Number.isSafeInteger(total) &&
    total <= 1_000_000_000_000 &&
    cashMinor !== null &&
    cashMinor >= total &&
    !busy;

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!canSell || cashMinor === null) return;
    setBusy(true);
    setError("");
    requestKey.current ??= crypto.randomUUID();
    try {
      const { sale } = await checkout(current.id, {
        storeId,
        idempotencyKey: requestKey.current,
        items: cart.map(({ item, quantity }) => ({
          variantId: item.variantId,
          quantity,
        })),
        cashReceivedMinor: cashMinor,
      });
      router.push(workspacePath(current.slug, `/sales/${sale.id}`));
    } catch (e) {
      // Keep the same key for retry if the network failed after the sale committed.
      setError(e instanceof Error ? e.message : "Checkout failed. Try again.");
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-10">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="text-xs font-semibold tracking-[0.22em] text-accent uppercase">
            Point of sale
          </p>
          <h1 className="mt-2 text-3xl font-semibold tracking-tight">
            Cash checkout
          </h1>
          <p className="mt-1 text-sm text-muted">
            Search or scan a product, take cash, and open a receipt.
          </p>
        </div>
        <Link
          href={workspacePath(current.slug, "/sales/history")}
          className="rounded-xl border border-line px-4 py-2.5 text-sm font-medium hover:border-accent/50"
        >
          Sales history
        </Link>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}
      <label className="block max-w-xs text-sm font-medium">
        Selling from
        <select
          aria-label="Selling store"
          className={`${field} mt-2`}
          value={storeId}
          onChange={(event) => changeStore(event.target.value)}
          disabled={busy}
        >
          {!storeId && <option value="">Select a store</option>}
          {stores
            .filter((store) => store.accessible)
            .map((store) => (
              <option key={store.id} value={store.id}>
                {store.name}
                {store.isDefault ? " (default)" : ""}
              </option>
            ))}
        </select>
        <span className="mt-1 block text-xs text-muted">
          Cart clears when you switch stores so stock is never mixed.
        </span>
      </label>
      {storeError && (
        <p role="alert" className="text-sm text-red-300">
          {storeError}
        </p>
      )}
      {selectedStore && (
        <p className="text-sm text-muted">
          Current selling location: {selectedStore.name}
        </p>
      )}

      <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(340px,420px)]">
        <section className="space-y-4" aria-label="Product search">
          <form onSubmit={(e) => void scan(e)}>
            <label className="block text-sm font-medium">
              Search name, SKU or barcode
              <span className="relative mt-2 block">
                <Search className="absolute top-3 left-3 h-4 w-4 text-muted" />
                <input
                  ref={searchRef}
                  autoFocus
                  className={`${field} pl-10`}
                  placeholder="Scan a barcode or type to search"
                  value={search}
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setLoading(true);
                  }}
                />
              </span>
            </label>
          </form>
          <div className="space-y-2">
            {loading ? (
              <p className="text-sm text-muted">Loading products…</p>
            ) : results.length === 0 ? (
              <div className="rounded-2xl border border-line bg-surface p-8 text-center text-sm text-muted">
                {search
                  ? "No matching stocked product."
                  : "No products available yet."}{" "}
                <Link
                  href={workspacePath(current.slug, "/inventory")}
                  className="text-accent hover:underline"
                >
                  View inventory
                </Link>
              </div>
            ) : (
              results.map((item) => (
                <button
                  key={item.variantId}
                  type="button"
                  disabled={item.quantity === 0}
                  onClick={() => add(item)}
                  className="flex w-full items-center justify-between gap-4 rounded-2xl border border-line bg-surface p-4 text-left transition-colors hover:border-accent/40 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <span className="min-w-0">
                    <span className="block truncate font-medium">
                      {item.productName}
                      {item.variantName !== "Standard" &&
                        ` · ${item.variantName}`}
                    </span>
                    <span className="mt-1 block truncate font-mono text-xs text-muted">
                      {item.sku}
                      {item.barcode && ` · ${item.barcode}`}
                    </span>
                    <span className="mt-1 block text-xs text-muted">
                      {item.quantity} available
                    </span>
                  </span>
                  <span className="shrink-0 text-right">
                    <span className="block font-semibold text-accent">
                      {formatMoney(item.priceMinor, current.currencyCode)}
                    </span>
                    <span className="mt-1 block text-xs text-muted">
                      {item.quantity ? "Add to cart" : "Out of stock"}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>
        </section>

        <form
          onSubmit={(e) => void submit(e)}
          className="rounded-2xl border border-line bg-surface p-5 lg:sticky lg:top-20"
        >
          <h2 className="flex items-center gap-2 text-lg font-semibold">
            <ShoppingCart className="h-5 w-5 text-accent" /> Cart
            <span className="ml-auto text-xs font-normal text-muted">
              {cart.length} line(s)
            </span>
          </h2>
          {cart.length === 0 ? (
            <p className="py-10 text-center text-sm text-muted">
              Scan or select a stocked product to start.
            </p>
          ) : (
            <div className="mt-4 divide-y divide-line border-t border-line">
              {cart.map(({ item, quantity }) => (
                <div key={item.variantId} className="py-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">
                        {item.productName}
                        {item.variantName !== "Standard" &&
                          ` · ${item.variantName}`}
                      </p>
                      <p className="mt-1 font-mono text-xs text-muted">
                        {item.sku}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => remove(item.variantId)}
                      aria-label={`Remove ${item.sku}`}
                      className="text-muted hover:text-red-300"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                  <div className="mt-3 flex items-center justify-between gap-2">
                    <div className="flex items-center rounded-lg border border-line">
                      <button
                        type="button"
                        disabled={quantity <= 1}
                        onClick={() =>
                          changeQuantity(item.variantId, quantity - 1)
                        }
                        aria-label={`Decrease ${item.sku}`}
                        className="p-2 disabled:opacity-30"
                      >
                        <Minus className="h-3 w-3" />
                      </button>
                      <input
                        type="number"
                        min="1"
                        max={item.quantity}
                        value={quantity}
                        aria-label={`${item.sku} quantity`}
                        onChange={(e) =>
                          changeQuantity(item.variantId, Number(e.target.value))
                        }
                        className="w-12 bg-transparent text-center text-sm outline-none"
                      />
                      <button
                        type="button"
                        disabled={quantity >= item.quantity}
                        onClick={() =>
                          changeQuantity(item.variantId, quantity + 1)
                        }
                        aria-label={`Increase ${item.sku}`}
                        className="p-2 disabled:opacity-30"
                      >
                        <Plus className="h-3 w-3" />
                      </button>
                    </div>
                    <span className="text-sm font-medium">
                      {formatMoney(
                        item.priceMinor * quantity,
                        current.currencyCode,
                      )}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
          <div className="space-y-2 border-t border-line pt-4 text-sm">
            <div className="flex justify-between text-muted">
              <span>Subtotal</span>
              <span>{formatMoney(total, current.currencyCode)}</span>
            </div>
            <div className="flex justify-between text-base font-semibold">
              <span>Total</span>
              <span>{formatMoney(total, current.currencyCode)}</span>
            </div>
          </div>
          <div className="mt-5 space-y-3">
            <label className="block text-sm font-medium">
              Cash received ({current.currencyCode})
              <input
                className={`${field} mt-2`}
                inputMode="decimal"
                placeholder="0.00"
                value={cash}
                onChange={(e) => {
                  setCash(e.target.value);
                  requestKey.current = null;
                }}
              />
            </label>
            <button
              type="button"
              disabled={cart.length === 0}
              onClick={() => {
                setCash(
                  (total / 10 ** minorDigits(current.currencyCode)).toFixed(
                    minorDigits(current.currencyCode),
                  ),
                );
                requestKey.current = null;
              }}
              className="text-xs text-accent hover:underline disabled:opacity-40"
            >
              Exact cash
            </button>
            <div className="flex justify-between border-t border-line pt-3 text-sm">
              <span className="text-muted">Change due</span>
              <span className="font-semibold">
                {change !== null && change >= 0
                  ? formatMoney(change, current.currencyCode)
                  : "—"}
              </span>
            </div>
          </div>
          <button
            type="submit"
            disabled={!canSell}
            className={`${button} mt-5 w-full`}
          >
            {busy ? "Completing sale…" : "Confirm cash sale"}
          </button>
          <p className="mt-3 text-xs text-muted">
            Prices and stock are checked again when you confirm. The receipt
            shows the final charged amounts.
          </p>
        </form>
      </div>
    </div>
  );
}
