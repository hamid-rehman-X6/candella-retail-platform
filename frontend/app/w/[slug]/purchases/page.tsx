"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { Search, Trash2 } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { Select } from "@/components/ui/select";
import { useStoreSelection } from "@/lib/use-store-selection";
import { listStock, type StockItem } from "@/lib/catalog-api";
import {
  listSuppliers,
  listSupplierSources,
  type Supplier,
  type SupplierSource,
} from "@/lib/suppliers-api";
import {
  createPurchase,
  listPurchases,
  type PurchaseSummary,
} from "@/lib/purchases-api";
import { formatDateTime } from "@/lib/datetime";
import { formatMoney, minorDigits, parseMoney } from "@/lib/money";

type Line = { item: StockItem; quantity: string; cost: string };
const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "cursor-pointer rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50";

export default function PurchasesPage() {
  const { current } = useApp();
  return <PurchasesWorkspace key={current.id} />;
}

function PurchasesWorkspace() {
  const { current } = useApp();
  const router = useRouter();
  const {
    stores,
    storeId,
    select,
    error: storeError,
  } = useStoreSelection(current.id);
  const manager = current.role === "owner" || current.role === "admin";
  const [supplierSearch, setSupplierSearch] = useState("");
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [supplierId, setSupplierId] = useState("");
  const [sources, setSources] = useState<SupplierSource[]>([]);
  const [productSearch, setProductSearch] = useState("");
  const [products, setProducts] = useState<StockItem[]>([]);
  const [lines, setLines] = useState<Line[]>([]);
  const [orders, setOrders] = useState<PurchaseSummary[]>([]);
  const [offset, setOffset] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [listLoading, setListLoading] = useState(false);
  const requestKey = useRef<string | null>(null);

  useEffect(() => {
    const timer = setTimeout(() => {
      const params = new URLSearchParams(window.location.search);
      const sku = params.get("sku");
      const id = params.get("supplierId");
      if (sku) setProductSearch(sku);
      if (id) {
        setSupplierId(id);
        setSupplierSearch(params.get("supplierName") ?? "");
      }
    }, 0);
    return () => clearTimeout(timer);
  }, []);

  useEffect(() => {
    let cancelled = false;
    const timer = setTimeout(() => {
      void listSuppliers(current.id, supplierSearch)
        .then(({ suppliers }) => {
          if (!cancelled)
            setSuppliers(suppliers.filter((s) => s.status === "active"));
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
  }, [current.id, supplierSearch]);
  useEffect(() => {
    if (!supplierId) return;
    let cancelled = false;
    void listSupplierSources(current.id, supplierId)
      .then(({ sources }) => {
        if (!cancelled) setSources(sources);
      })
      .catch(() => {
        if (!cancelled) setSources([]);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id, supplierId]);
  useEffect(() => {
    if (!storeId) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      void listStock(current.id, storeId, productSearch)
        .then(({ items }) => {
          if (!cancelled) setProducts(items);
        })
        .catch((e) => {
          if (!cancelled)
            setError(
              e instanceof Error ? e.message : "Could not load products.",
            );
        });
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, storeId, productSearch]);
  useEffect(() => {
    if (!storeId) return;
    let cancelled = false;
    const timer = setTimeout(() => {
      setListLoading(true);
      void listPurchases(current.id, storeId, offset)
        .then(({ orders }) => {
          if (!cancelled) setOrders(orders);
        })
        .catch((e) => {
          if (!cancelled)
            setError(
              e instanceof Error
                ? e.message
                : "Could not load purchase orders.",
            );
        })
        .finally(() => {
          if (!cancelled) setListLoading(false);
        });
    }, 0);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, storeId, offset]);

  function changeStore(id: string) {
    select(id);
    setLines([]);
    setProducts([]);
    setProductSearch("");
    setOffset(0);
    requestKey.current = null;
  }
  function add(item: StockItem) {
    if (lines.some((line) => line.item.variantId === item.variantId)) return;
    const source = sources.find(
      (source) => source.variantId === item.variantId,
    );
    const cost =
      source?.costMinor == null
        ? ""
        : (source.costMinor / 10 ** minorDigits(current.currencyCode)).toFixed(
            minorDigits(current.currencyCode),
          );
    setLines((old) => [...old, { item, quantity: "1", cost }]);
    setProductSearch("");
    requestKey.current = null;
  }
  function changeLine(
    id: string,
    values: Partial<Pick<Line, "quantity" | "cost">>,
  ) {
    setLines((old) =>
      old.map((line) =>
        line.item.variantId === id ? { ...line, ...values } : line,
      ),
    );
    requestKey.current = null;
  }
  const valid =
    lines.length > 0 &&
    lines.every((line) => {
      const quantity = Number(line.quantity),
        cost = parseMoney(line.cost, current.currencyCode);
      return (
        Number.isSafeInteger(quantity) &&
        quantity > 0 &&
        quantity <= 1_000_000 &&
        cost !== null &&
        cost * quantity <= 1_000_000_000_000
      );
    });
  const total = lines.reduce(
    (sum, line) =>
      sum +
      (parseMoney(line.cost, current.currencyCode) ?? 0) *
        Number(line.quantity || 0),
    0,
  );
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (
      !valid ||
      !supplierId ||
      !storeId ||
      !Number.isSafeInteger(total) ||
      total > 1_000_000_000_000 ||
      busy
    )
      return;
    setBusy(true);
    setError("");
    requestKey.current ??= crypto.randomUUID();
    try {
      const { order } = await createPurchase(current.id, {
        storeId,
        supplierId,
        idempotencyKey: requestKey.current,
        items: lines.map((line) => ({
          variantId: line.item.variantId,
          quantity: Number(line.quantity),
          unitCostMinor: parseMoney(line.cost, current.currencyCode)!,
        })),
      });
      router.push(workspacePath(current.slug, `/purchases/${order.id}`));
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "Could not create purchase order.",
      );
      setBusy(false);
    }
  }

  if (!manager)
    return (
      <div className="mx-auto max-w-5xl p-6 text-sm text-muted">
        Ask a workspace owner for purchase order access.
      </div>
    );
  return (
    <div className="mx-auto max-w-7xl space-y-6 pb-12">
      <div>
        <h1 className="text-3xl font-semibold">Purchases</h1>
        <p className="mt-1 text-sm text-muted">
          Order stock from a supplier, approve it, then receive each delivery
          once.
        </p>
      </div>
      {(error || storeError) && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error || storeError}
        </p>
      )}
      <label className="block max-w-xs text-sm font-medium">
        Store
        <Select
          ariaLabel="Purchase store"
          className={`${field} mt-2`}
          value={storeId}
          onValueChange={changeStore}
          options={stores
            .filter((store) => store.accessible)
            .map((store) => ({ value: store.id, label: store.name }))}
          disabled={busy}
        />
      </label>
      {storeId && (
        <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(340px,430px)]">
          <section className="space-y-4 rounded-2xl border border-line bg-surface p-5">
            <h2 className="text-lg font-semibold">New purchase order</h2>
            <label className="block text-sm font-medium">
              Find supplier
              <input
                className={`${field} mt-2`}
                placeholder="Search suppliers"
                value={supplierSearch}
                onChange={(e) => setSupplierSearch(e.target.value)}
                disabled={busy}
              />
            </label>
            <Select
              ariaLabel="Supplier"
              className={field}
              value={supplierId}
              disabled={busy}
              onValueChange={(id) => {
                setSupplierId(id);
                setSources([]);
                requestKey.current = null;
              }}
              options={[
                { value: "", label: "Select supplier" },
                ...[
                  ...(supplierId && !suppliers.some((s) => s.id === supplierId)
                    ? [{ id: supplierId, name: "Selected supplier" }]
                    : []),
                  ...suppliers,
                ].map((s) => ({ value: s.id, label: s.name })),
              ]}
            />
            <label className="block text-sm font-medium">
              Search product or SKU
              <span className="relative mt-2 block">
                <Search className="absolute top-3 left-3 h-4 w-4 text-muted" />
                <input
                  className={`${field} pl-10`}
                  placeholder="Product name, SKU or barcode"
                  value={productSearch}
                  onChange={(e) => setProductSearch(e.target.value)}
                  disabled={busy}
                />
              </span>
            </label>
            <div className="max-h-64 space-y-2 overflow-y-auto">
              {products
                .filter(
                  (item) =>
                    !lines.some(
                      (line) => line.item.variantId === item.variantId,
                    ),
                )
                .map((item) => (
                  <button
                    key={item.variantId}
                    type="button"
                    onClick={() => add(item)}
                    className="flex w-full cursor-pointer justify-between gap-3 rounded-xl border border-line p-3 text-left text-sm hover:border-accent/50"
                  >
                    <span>
                      {item.productName} · {item.variantName}
                      <span className="block font-mono text-xs text-muted">
                        {item.sku}
                      </span>
                    </span>
                    <span className="shrink-0 text-accent-ink">Add</span>
                  </button>
                ))}
              {products.length === 0 && (
                <p className="text-sm text-muted">
                  No matching active products.
                </p>
              )}
            </div>
            {lines.length > 0 && (
              <form
                onSubmit={(e) => void submit(e)}
                className="space-y-3 border-t border-line pt-4"
              >
                <h3 className="font-semibold">Order lines</h3>
                {lines.map((line) => (
                  <div
                    key={line.item.variantId}
                    className="grid gap-2 rounded-xl border border-line p-3 sm:grid-cols-[minmax(0,1fr)_80px_110px_32px] sm:items-end"
                  >
                    <span className="min-w-0 text-sm">
                      <span className="block truncate font-medium">
                        {line.item.productName} · {line.item.variantName}
                      </span>
                      <span className="font-mono text-xs text-muted">
                        {line.item.sku}
                      </span>
                    </span>
                    <label className="text-xs text-muted">
                      Units
                      <input
                        className={`${field} mt-1`}
                        type="number"
                        min="1"
                        max="1000000"
                        step="1"
                        value={line.quantity}
                        onChange={(e) =>
                          changeLine(line.item.variantId, {
                            quantity: e.target.value,
                          })
                        }
                      />
                    </label>
                    <label className="text-xs text-muted">
                      Unit cost
                      <input
                        className={`${field} mt-1`}
                        inputMode="decimal"
                        placeholder="0.00"
                        value={line.cost}
                        onChange={(e) =>
                          changeLine(line.item.variantId, {
                            cost: e.target.value,
                          })
                        }
                      />
                    </label>
                    <button
                      type="button"
                      onClick={() => {
                        setLines((old) =>
                          old.filter(
                            (other) =>
                              other.item.variantId !== line.item.variantId,
                          ),
                        );
                        requestKey.current = null;
                      }}
                      aria-label={`Remove ${line.item.sku}`}
                      className="cursor-pointer rounded-lg p-2 text-danger-ink"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
                <p className="text-right text-sm font-semibold">
                  Order total:{" "}
                  {Number.isFinite(total)
                    ? formatMoney(total, current.currencyCode)
                    : "—"}
                </p>
                <button
                  type="submit"
                  disabled={
                    !valid ||
                    !supplierId ||
                    busy ||
                    !Number.isSafeInteger(total) ||
                    total > 1_000_000_000_000
                  }
                  className={button}
                >
                  {busy ? "Creating…" : "Create draft order"}
                </button>
                <p className="text-xs text-muted">
                  A draft does not change stock. Review and approve it before
                  recording a delivery.
                </p>
              </form>
            )}
          </section>
          <section className="space-y-4 rounded-2xl border border-line bg-surface p-5">
            <h2 className="text-lg font-semibold">Purchase orders</h2>
            {listLoading ? (
              <p className="text-sm text-muted">Loading orders…</p>
            ) : orders.length === 0 ? (
              <p className="text-sm text-muted">
                No purchase orders for this store yet.
              </p>
            ) : (
              orders.map((order) => (
                <Link
                  key={order.id}
                  href={workspacePath(current.slug, `/purchases/${order.id}`)}
                  className="block rounded-xl border border-line p-3 hover:border-accent/50"
                >
                  <span className="flex justify-between gap-2 text-sm font-medium">
                    <span>
                      {order.number} · {order.supplierName}
                    </span>
                    <span>
                      {formatMoney(order.totalCostMinor, order.currencyCode)}
                    </span>
                  </span>
                  <span className="mt-1 block text-xs text-muted">
                    {order.status.replaceAll("_", " ")} ·{" "}
                    {formatDateTime(order.createdAt, current.timezone)}
                  </span>
                </Link>
              ))
            )}
            <div className="flex justify-between text-sm">
              <button
                type="button"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - 50))}
                className="cursor-pointer disabled:opacity-40"
              >
                Previous
              </button>
              <button
                type="button"
                disabled={orders.length < 50}
                onClick={() => setOffset(offset + 50)}
                className="cursor-pointer disabled:opacity-40"
              >
                Next
              </button>
            </div>
          </section>
        </div>
      )}
    </div>
  );
}
