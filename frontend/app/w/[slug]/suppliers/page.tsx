"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { Archive, Pencil, Plus, Search, Truck } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import { useStoreSelection } from "@/lib/use-store-selection";
import {
  formatMoney,
  listStock,
  minorDigits,
  parseMoney,
  type StockItem,
} from "@/lib/catalog-api";
import {
  createSupplier,
  listSupplierSources,
  listSuppliers,
  saveSupplierSource,
  updateSupplier,
  type Supplier,
  type SupplierInput,
  type SupplierSource,
} from "@/lib/suppliers-api";
import { toast } from "@/lib/toast";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:opacity-50";
const emptySupplier: SupplierInput = {
  name: "",
  phone: "",
  email: "",
  address: "",
};

export default function SuppliersPage() {
  const { current } = useApp();
  return <SuppliersWorkspacePage key={current.id} />;
}

function SuppliersWorkspacePage() {
  const { current } = useApp();
  const { storeId } = useStoreSelection(current.id);
  const editable = current.role === "owner" || current.role === "admin";
  const [search, setSearch] = useState("");
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [loading, setLoading] = useState(true);
  const [hasMore, setHasMore] = useState(false);
  const [selected, setSelected] = useState<Supplier | null>(null);
  const [sources, setSources] = useState<SupplierSource[]>([]);
  const [sourceHasMore, setSourceHasMore] = useState(false);
  const [sourceLoading, setSourceLoading] = useState(false);
  const [error, setError] = useState("");
  const [supplierFormOpen, setSupplierFormOpen] = useState(false);
  const [editingSupplierId, setEditingSupplierId] = useState<string | null>(
    null,
  );
  const [supplierForm, setSupplierForm] =
    useState<SupplierInput>(emptySupplier);
  const [supplierBusy, setSupplierBusy] = useState(false);
  const [variantSearch, setVariantSearch] = useState("");
  const [variants, setVariants] = useState<StockItem[]>([]);
  const [variantId, setVariantId] = useState("");
  const [supplierSku, setSupplierSku] = useState("");
  const [cost, setCost] = useState("");
  const [sourceBusy, setSourceBusy] = useState(false);
  const supplierFetchId = useRef(0);
  const sourceFetchId = useRef(0);
  const selectedId = selected?.id;
  const selectedStatus = selected?.status;

  const refreshSuppliers = useCallback(async () => {
    const requestId = ++supplierFetchId.current;
    setLoading(true);
    try {
      const result = await listSuppliers(current.id, search);
      if (requestId !== supplierFetchId.current) return;
      setSuppliers(result.suppliers);
      setHasMore(result.suppliers.length === 100);
      setError("");
    } catch (e) {
      if (requestId === supplierFetchId.current)
        setError(e instanceof Error ? e.message : "Could not load suppliers.");
    } finally {
      if (requestId === supplierFetchId.current) setLoading(false);
    }
  }, [current.id, search]);

  useEffect(() => {
    const timer = setTimeout(() => void refreshSuppliers(), 200);
    return () => clearTimeout(timer);
  }, [refreshSuppliers]);

  const refreshSources = useCallback(
    async (supplierId: string) => {
      const requestId = ++sourceFetchId.current;
      setSourceLoading(true);
      try {
        const result = await listSupplierSources(current.id, supplierId);
        if (requestId !== sourceFetchId.current) return;
        setSources(result.sources);
        setSourceHasMore(result.sources.length === 100);
        setError("");
      } catch (e) {
        if (requestId === sourceFetchId.current)
          setError(e instanceof Error ? e.message : "Could not load sourcing.");
      } finally {
        if (requestId === sourceFetchId.current) setSourceLoading(false);
      }
    },
    [current.id],
  );

  useEffect(() => {
    if (!selectedId) return;
    const timer = setTimeout(() => void refreshSources(selectedId), 0);
    return () => clearTimeout(timer);
  }, [selectedId, refreshSources]);

  useEffect(() => {
    if (
      !storeId ||
      !selectedId ||
      selectedStatus !== "active" ||
      !variantSearch.trim()
    )
      return;
    let cancelled = false;
    const timer = setTimeout(async () => {
      try {
        const result = await listStock(current.id, storeId, variantSearch);
        if (!cancelled) setVariants(result.items);
      } catch (e) {
        if (!cancelled)
          setError(
            e instanceof Error ? e.message : "Could not search products.",
          );
      }
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [current.id, storeId, selectedId, selectedStatus, variantSearch]);

  function clearSourceForm() {
    setVariantId("");
    setVariantSearch("");
    setSupplierSku("");
    setCost("");
    setVariants([]);
  }

  function selectSupplier(supplier: Supplier) {
    sourceFetchId.current++;
    setSelected(supplier);
    setSources([]);
    setSourceLoading(true);
    clearSourceForm();
    setError("");
  }

  function startSupplierForm(supplier?: Supplier) {
    setEditingSupplierId(supplier?.id ?? null);
    setSupplierForm(
      supplier
        ? {
            name: supplier.name,
            phone: supplier.phone,
            email: supplier.email,
            address: supplier.address,
          }
        : emptySupplier,
    );
    setSupplierFormOpen(true);
    setError("");
  }

  async function saveSupplier(e: FormEvent) {
    e.preventDefault();
    if (!supplierForm.name.trim() || supplierBusy) return;
    setSupplierBusy(true);
    setError("");
    try {
      const { supplier } = editingSupplierId
        ? await updateSupplier(current.id, editingSupplierId, supplierForm)
        : await createSupplier(current.id, supplierForm);
      setSupplierFormOpen(false);
      sourceFetchId.current++;
      setSelected(supplier);
      setSources([]);
      setSourceLoading(true);
      toast.success(editingSupplierId ? "Supplier updated" : "Supplier added");
      await refreshSuppliers();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save supplier.");
    } finally {
      setSupplierBusy(false);
    }
  }

  async function changeStatus(supplier: Supplier) {
    const next = supplier.status === "active" ? "archived" : "active";
    if (
      next === "archived" &&
      !window.confirm(
        `Archive ${supplier.name}? Existing product links will remain visible.`,
      )
    )
      return;
    setError("");
    try {
      const result = await updateSupplier(current.id, supplier.id, {
        status: next,
      });
      setSelected(result.supplier);
      clearSourceForm();
      toast.success(
        next === "archived" ? "Supplier archived" : "Supplier restored",
      );
      await refreshSuppliers();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not update supplier.");
    }
  }

  function editSource(source: SupplierSource) {
    setVariantId(source.variantId);
    setVariantSearch(source.sku);
    setSupplierSku(source.supplierSku);
    setCost(
      source.costMinor === null
        ? ""
        : String(source.costMinor / 10 ** minorDigits(current.currencyCode)),
    );
    setError("");
  }

  async function saveSource(e: FormEvent) {
    e.preventDefault();
    if (!selected || !variantId || sourceBusy) return;
    const costMinor =
      cost.trim() === "" ? null : parseMoney(cost, current.currencyCode);
    if (cost.trim() !== "" && costMinor === null) {
      setError("Enter a valid purchase cost, or leave it blank if unknown.");
      return;
    }
    setSourceBusy(true);
    setError("");
    try {
      await saveSupplierSource(current.id, selected.id, {
        variantId,
        supplierSku,
        costMinor,
      });
      toast.success("Product sourcing saved");
      clearSourceForm();
      await refreshSources(selected.id);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save sourcing.");
    } finally {
      setSourceBusy(false);
    }
  }

  return (
    <div className="space-y-6 pb-10">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold">Suppliers</h1>
          <p className="mt-1 text-sm text-muted">
            Keep supplier contacts and purchase costs linked to product
            variants.
          </p>
        </div>
        {editable && (
          <button onClick={() => startSupplierForm()} className={button}>
            <Plus className="mr-2 inline h-4 w-4" /> Add supplier
          </button>
        )}
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}

      {supplierFormOpen && editable && (
        <form
          onSubmit={(e) => void saveSupplier(e)}
          className="grid gap-3 rounded-2xl border border-line bg-surface p-5 sm:grid-cols-2"
        >
          <h2 className="text-lg font-semibold sm:col-span-2">
            {editingSupplierId ? "Edit supplier" : "New supplier"}
          </h2>
          {(["name", "phone", "email", "address"] as const).map((key) => (
            <label
              key={key}
              className={`text-sm ${key === "address" ? "sm:col-span-2" : ""}`}
            >
              <span className="capitalize">{key}</span>
              {key === "name" && " *"}
              <input
                className={`${field} mt-1`}
                type={key === "email" ? "email" : "text"}
                required={key === "name"}
                maxLength={
                  { name: 160, phone: 40, email: 254, address: 500 }[key]
                }
                value={supplierForm[key]}
                onChange={(e) =>
                  setSupplierForm((old) => ({ ...old, [key]: e.target.value }))
                }
              />
            </label>
          ))}
          <div className="flex gap-2 sm:col-span-2">
            <button type="submit" disabled={supplierBusy} className={button}>
              {supplierBusy ? "Saving…" : "Save supplier"}
            </button>
            <button
              type="button"
              onClick={() => setSupplierFormOpen(false)}
              className="rounded-xl border border-line px-4 py-2.5 text-sm"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      <div className="grid gap-5 xl:grid-cols-[minmax(280px,360px)_minmax(0,1fr)]">
        <section className="space-y-3">
          <label className="relative block">
            <span className="sr-only">Search suppliers</span>
            <Search className="absolute top-3 left-3 h-4 w-4 text-muted" />
            <input
              className={`${field} pl-9`}
              placeholder="Search supplier, phone or email"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          {loading ? (
            <p className="text-sm text-muted">Loading suppliers…</p>
          ) : suppliers.length === 0 ? (
            <div className="rounded-2xl border border-line bg-surface p-6 text-sm text-muted">
              {search
                ? "No matching suppliers."
                : "No suppliers yet. Add one to record where you buy stock."}
            </div>
          ) : (
            <div className="space-y-2">
              {suppliers.map((supplier) => (
                <button
                  key={supplier.id}
                  type="button"
                  onClick={() => selectSupplier(supplier)}
                  className={`w-full rounded-2xl border p-4 text-left transition-colors ${selected?.id === supplier.id ? "border-accent bg-accent/10" : "border-line bg-surface hover:border-line-strong"}`}
                >
                  <span className="flex items-center justify-between gap-2">
                    <span className="font-medium">{supplier.name}</span>
                    {supplier.status === "archived" && (
                      <span className="text-xs text-muted">Archived</span>
                    )}
                  </span>
                  <span className="mt-1 block text-xs text-muted">
                    {supplier.phone || supplier.email || "No contact details"}
                  </span>
                </button>
              ))}
            </div>
          )}
          {hasMore && (
            <button
              type="button"
              className="rounded-xl border border-line px-4 py-2 text-sm"
              onClick={async () => {
                try {
                  const result = await listSuppliers(
                    current.id,
                    search,
                    suppliers.length,
                  );
                  setSuppliers((old) => [...old, ...result.suppliers]);
                  setHasMore(result.suppliers.length === 100);
                } catch (e) {
                  setError(
                    e instanceof Error
                      ? e.message
                      : "Could not load more suppliers.",
                  );
                }
              }}
            >
              Load more
            </button>
          )}
        </section>

        <section className="rounded-2xl border border-line bg-surface p-5">
          {!selected ? (
            <div className="flex min-h-64 flex-col items-center justify-center text-center text-muted">
              <Truck className="mb-3 h-8 w-8" />
              <p>Select a supplier to see contacts and linked products.</p>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h2 className="text-xl font-semibold">{selected.name}</h2>
                  <p className="mt-1 text-sm text-muted">
                    {selected.phone || "No phone"} ·{" "}
                    {selected.email || "No email"}
                  </p>
                  {selected.address && (
                    <p className="mt-1 text-sm text-muted">
                      {selected.address}
                    </p>
                  )}
                  {selected.status === "archived" && (
                    <p className="mt-2 text-xs text-amber-400">
                      Archived: existing links remain visible; new links are
                      disabled.
                    </p>
                  )}
                </div>
                {editable && (
                  <div className="flex gap-2">
                    <button
                      type="button"
                      onClick={() => startSupplierForm(selected)}
                      className="rounded-lg border border-line px-3 py-2 text-xs"
                    >
                      <Pencil className="mr-1 inline h-3.5 w-3.5" />
                      Edit
                    </button>
                    <button
                      type="button"
                      onClick={() => void changeStatus(selected)}
                      className="rounded-lg border border-line px-3 py-2 text-xs"
                    >
                      <Archive className="mr-1 inline h-3.5 w-3.5" />
                      {selected.status === "active" ? "Archive" : "Restore"}
                    </button>
                  </div>
                )}
              </div>

              {editable && selected.status === "active" && (
                <form
                  onSubmit={(e) => void saveSource(e)}
                  className="space-y-3 rounded-xl border border-line bg-canvas p-4"
                >
                  <h3 className="font-medium">Link a product variant</h3>
                  <p className="text-xs text-muted">
                    Search by product name, SKU or barcode. Purchase cost is per
                    individual unit.
                  </p>
                  <label className="block text-sm">
                    Product / variant
                    <input
                      className={`${field} mt-1`}
                      placeholder="Search product or SKU"
                      value={variantSearch}
                      onChange={(e) => {
                        setVariantSearch(e.target.value);
                        setVariantId("");
                      }}
                    />
                  </label>
                  {variantSearch && (
                    <select
                      className={field}
                      value={variantId}
                      onChange={(e) => setVariantId(e.target.value)}
                      aria-label="Choose product variant"
                      required
                    >
                      <option value="">Choose a matching variant</option>
                      {variantId &&
                        !variants.some(
                          (item) => item.variantId === variantId,
                        ) &&
                        sources.find(
                          (item) => item.variantId === variantId,
                        ) && (
                          <option value={variantId}>
                            {
                              sources.find(
                                (item) => item.variantId === variantId,
                              )?.productName
                            }{" "}
                            ·{" "}
                            {
                              sources.find(
                                (item) => item.variantId === variantId,
                              )?.variantName
                            }
                          </option>
                        )}
                      {variants.map((item) => (
                        <option key={item.variantId} value={item.variantId}>
                          {item.productName} · {item.variantName} · {item.sku}
                        </option>
                      ))}
                    </select>
                  )}
                  <div className="grid gap-3 sm:grid-cols-2">
                    <label className="text-sm">
                      Supplier SKU (optional)
                      <input
                        className={`${field} mt-1`}
                        maxLength={80}
                        value={supplierSku}
                        onChange={(e) => setSupplierSku(e.target.value)}
                      />
                    </label>
                    <label className="text-sm">
                      Purchase cost per unit (optional)
                      <input
                        className={`${field} mt-1`}
                        inputMode="decimal"
                        placeholder="Unknown"
                        value={cost}
                        onChange={(e) => setCost(e.target.value)}
                      />
                    </label>
                  </div>
                  <button
                    type="submit"
                    disabled={!variantId || sourceBusy}
                    className={button}
                  >
                    {sourceBusy ? "Saving…" : "Save sourcing"}
                  </button>
                </form>
              )}

              <div>
                <h3 className="font-semibold">Linked products</h3>
                {sourceLoading ? (
                  <p className="mt-3 text-sm text-muted">Loading links…</p>
                ) : sources.length === 0 ? (
                  <p className="mt-3 text-sm text-muted">
                    No product variants linked yet.
                  </p>
                ) : (
                  <div className="mt-3 divide-y divide-line rounded-xl border border-line">
                    {sources.map((source) => (
                      <div
                        key={source.variantId}
                        className="flex flex-wrap items-center justify-between gap-3 p-4 text-sm"
                      >
                        <div>
                          <p className="font-medium">
                            {source.productName} · {source.variantName}
                          </p>
                          <p className="mt-1 text-xs text-muted">
                            {source.sku}
                            {source.supplierSku &&
                              ` · Supplier SKU: ${source.supplierSku}`}
                            {source.productStatus === "archived" &&
                              " · Product archived"}
                          </p>
                        </div>
                        <div className="flex items-center gap-3">
                          <span>
                            {source.costMinor === null
                              ? "Cost unknown"
                              : formatMoney(
                                  source.costMinor,
                                  current.currencyCode,
                                )}
                          </span>
                          {editable &&
                            selected.status === "active" &&
                            source.productStatus === "active" && (
                              <button
                                type="button"
                                onClick={() => editSource(source)}
                                className="text-xs text-accent hover:underline"
                              >
                                Edit
                              </button>
                            )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                {sourceHasMore && (
                  <button
                    type="button"
                    className="mt-3 rounded-xl border border-line px-4 py-2 text-sm"
                    onClick={async () => {
                      try {
                        const result = await listSupplierSources(
                          current.id,
                          selected.id,
                          sources.length,
                        );
                        setSources((old) => [...old, ...result.sources]);
                        setSourceHasMore(result.sources.length === 100);
                      } catch (e) {
                        setError(
                          e instanceof Error
                            ? e.message
                            : "Could not load more links.",
                        );
                      }
                    }}
                  >
                    Load more
                  </button>
                )}
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
