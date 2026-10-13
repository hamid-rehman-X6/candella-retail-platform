"use client";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { PackagePlus, Plus, Search, X } from "lucide-react";
import { useApp } from "@/components/app/app-context";
import {
  addVariant,
  createProduct,
  formatMoney,
  listProducts,
  minorDigits,
  parseMoney,
  updateProduct,
  updateVariant,
  type Product,
  type Variant,
} from "@/lib/catalog-api";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:opacity-50";

export default function ProductsPage() {
  const { current } = useApp();
  const editable = current.role === "owner" || current.role === "admin";
  const [products, setProducts] = useState<Product[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [form, setForm] = useState<
    "create" | "variant" | "variant_edit" | "edit" | null
  >(null);
  const [selected, setSelected] = useState<Product | null>(null);
  const [selectedVariant, setSelectedVariant] = useState<Variant | null>(null);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [variantName, setVariantName] = useState("");
  const [sku, setSku] = useState("");
  const [barcode, setBarcode] = useState("");
  const [price, setPrice] = useState("");
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);

  const refresh = useCallback(
    async (q = search) => {
      setLoading(true);
      setError("");
      try {
        const { products: next } = await listProducts(current.id, q);
        setProducts(next);
        setOffset(0);
        setHasMore(next.length === 100);
      } catch (e) {
        setError(e instanceof Error ? e.message : "Could not load products.");
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

  function openCreate() {
    setSelected(null);
    setName("");
    setDescription("");
    setVariantName("");
    setSku("");
    setBarcode("");
    setPrice("");
    setError("");
    setForm("create");
  }
  function openVariant(product: Product) {
    setSelected(product);
    setVariantName("");
    setSku("");
    setBarcode("");
    setPrice("");
    setError("");
    setForm("variant");
  }
  function openEdit(product: Product) {
    setSelected(product);
    setName(product.name);
    setDescription(product.description);
    setError("");
    setForm("edit");
  }
  function openVariantEdit(product: Product, variant: Variant) {
    setSelected(product);
    setSelectedVariant(variant);
    setVariantName(variant.name);
    setSku(variant.sku);
    setBarcode(variant.barcode ?? "");
    const digits = minorDigits(current.currencyCode);
    setPrice((variant.priceMinor / 10 ** digits).toFixed(digits));
    setError("");
    setForm("variant_edit");
  }
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!form) return;
    const priceMinor =
      form === "edit" ? 0 : parseMoney(price, current.currencyCode);
    if (priceMinor === null) {
      setError(`Enter a valid price for ${current.currencyCode}.`);
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (form === "create") {
        await createProduct(current.id, {
          name,
          description,
          variant: {
            name: variantName || "Standard",
            sku,
            barcode,
            priceMinor,
          },
        });
      } else if (form === "variant" && selected) {
        await addVariant(current.id, selected.id, {
          name: variantName || "Standard",
          sku,
          barcode,
          priceMinor,
        });
      } else if (form === "variant_edit" && selectedVariant) {
        await updateVariant(current.id, selectedVariant.id, {
          name: variantName,
          sku,
          barcode,
          priceMinor,
        });
      } else if (form === "edit" && selected) {
        await updateProduct(current.id, selected.id, { name, description });
      }
      setForm(null);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not save product.");
    } finally {
      setBusy(false);
    }
  }
  async function toggleStatus(product: Product) {
    if (
      !confirm(
        `${product.status === "active" ? "Archive" : "Restore"} ${product.name}?`,
      )
    )
      return;
    setError("");
    try {
      await updateProduct(current.id, product.id, {
        status: product.status === "active" ? "archived" : "active",
      });
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Could not update product.");
    }
  }
  async function loadMore() {
    const nextOffset = offset + 100;
    try {
      const { products: next } = await listProducts(
        current.id,
        search,
        nextOffset,
      );
      setProducts((previous) => [...previous, ...next]);
      setOffset(nextOffset);
      setHasMore(next.length === 100);
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "Could not load more products.",
      );
    }
  }

  return (
    <div className="space-y-7">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold">Products</h1>
          <p className="mt-1 text-sm text-muted">
            Your sellable catalog. Add a SKU for each size, color, or pack.
          </p>
        </div>
        {editable && (
          <button type="button" onClick={openCreate} className={button}>
            <PackagePlus size={17} /> Add product
          </button>
        )}
      </div>
      <label className="relative block max-w-xl">
        <span className="sr-only">Search products</span>
        <Search className="absolute top-3 left-3 text-muted" size={17} />
        <input
          className={`${field} pl-10`}
          placeholder="Search name, SKU or barcode"
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
        <p className="text-sm text-muted">Loading products…</p>
      ) : products.length === 0 ? (
        <div className="rounded-2xl border border-line bg-surface p-10 text-center">
          <PackagePlus className="mx-auto text-accent" size={28} />
          <h2 className="mt-4 font-semibold">
            {search ? "No matching products" : "No products yet"}
          </h2>
          <p className="mt-1 text-sm text-muted">
            {search
              ? "Try another name, SKU or barcode."
              : "Start with the products you sell most often."}
          </p>
          {!search && editable && (
            <button
              type="button"
              onClick={openCreate}
              className={`${button} mt-5`}
            >
              Add first product
            </button>
          )}
        </div>
      ) : (
        <div className="space-y-3">
          {products.map((product) => (
            <article
              key={product.id}
              className="rounded-2xl border border-line bg-surface p-5"
            >
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="font-semibold">{product.name}</h2>
                    {product.status === "archived" && (
                      <span className="rounded-full bg-white/10 px-2 py-0.5 text-xs text-muted">
                        Archived
                      </span>
                    )}
                  </div>
                  {product.description && (
                    <p className="mt-1 text-sm text-muted">
                      {product.description}
                    </p>
                  )}
                </div>
                {editable && (
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => openEdit(product)}
                      className="rounded-lg border border-line px-3 py-1.5 text-xs hover:border-line-strong"
                    >
                      Edit
                    </button>
                    {product.status === "active" && (
                      <button
                        type="button"
                        onClick={() => openVariant(product)}
                        className="rounded-lg border border-line px-3 py-1.5 text-xs hover:border-line-strong"
                      >
                        <Plus size={13} className="mr-1 inline" />
                        Variant
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => void toggleStatus(product)}
                      className="rounded-lg border border-line px-3 py-1.5 text-xs text-muted hover:border-line-strong"
                    >
                      {product.status === "active" ? "Archive" : "Restore"}
                    </button>
                  </div>
                )}
              </div>
              <div className="mt-4 divide-y divide-line border-t border-line text-sm">
                {product.variants.map((variant) => (
                  <div
                    key={variant.id}
                    className="grid gap-1 py-3 sm:grid-cols-[1fr_1fr_auto_auto] sm:gap-4"
                  >
                    <span>{variant.name}</span>
                    <span className="font-mono text-muted">
                      {variant.sku}
                      {variant.barcode && ` · ${variant.barcode}`}
                    </span>
                    <span>
                      {formatMoney(variant.priceMinor, current.currencyCode)}
                    </span>
                    <span
                      className={
                        variant.quantity === 0 ? "text-amber-400" : "text-muted"
                      }
                    >
                      {variant.quantity} in stock
                      {editable && product.status === "active" && (
                        <button
                          type="button"
                          onClick={() => openVariantEdit(product, variant)}
                          className="ml-3 text-xs text-accent hover:underline"
                        >
                          Edit variant
                        </button>
                      )}
                    </span>
                  </div>
                ))}
              </div>
            </article>
          ))}
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
      {form && (
        <div
          className="fixed inset-0 z-[70] flex items-center justify-center bg-black/75 p-4"
          role="presentation"
          onMouseDown={(e) => {
            if (e.target === e.currentTarget) setForm(null);
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-label={
              form === "create"
                ? "Add product"
                : form === "variant"
                  ? "Add variant"
                  : form === "variant_edit"
                    ? "Edit variant"
                    : "Edit product"
            }
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-line bg-canvas-raised p-6 shadow-2xl"
          >
            <div className="flex items-start justify-between">
              <h2 className="text-xl font-semibold">
                {form === "create"
                  ? "Add product"
                  : form === "variant"
                    ? `New variant for ${selected?.name}`
                    : form === "variant_edit"
                      ? `Edit ${selectedVariant?.name}`
                      : "Edit product"}
              </h2>
              <button
                type="button"
                onClick={() => setForm(null)}
                aria-label="Close"
                className="p-1 text-muted"
              >
                <X size={20} />
              </button>
            </div>
            <form onSubmit={(e) => void submit(e)} className="mt-6 space-y-4">
              {(form === "create" || form === "edit") && (
                <>
                  <label className="block text-sm">
                    Product name
                    <input
                      className={`${field} mt-1`}
                      required
                      maxLength={160}
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                    />
                  </label>
                  <label className="block text-sm">
                    Description <span className="text-muted">(optional)</span>
                    <textarea
                      className={`${field} mt-1`}
                      rows={2}
                      maxLength={2000}
                      value={description}
                      onChange={(e) => setDescription(e.target.value)}
                    />
                  </label>
                </>
              )}
              {form !== "edit" && (
                <>
                  <label className="block text-sm">
                    Variant name{" "}
                    <span className="text-muted">
                      (e.g. Blue / Large; optional)
                    </span>
                    <input
                      className={`${field} mt-1`}
                      maxLength={100}
                      value={variantName}
                      onChange={(e) => setVariantName(e.target.value)}
                    />
                  </label>
                  <label className="block text-sm">
                    SKU
                    <input
                      className={`${field} mt-1`}
                      required
                      maxLength={80}
                      value={sku}
                      onChange={(e) => setSku(e.target.value)}
                    />
                  </label>
                  <label className="block text-sm">
                    Barcode <span className="text-muted">(optional)</span>
                    <input
                      className={`${field} mt-1`}
                      maxLength={80}
                      value={barcode}
                      onChange={(e) => setBarcode(e.target.value)}
                    />
                  </label>
                  <label className="block text-sm">
                    Selling price ({current.currencyCode})
                    <input
                      className={`${field} mt-1`}
                      inputMode="decimal"
                      required
                      placeholder="0.00"
                      value={price}
                      onChange={(e) => setPrice(e.target.value)}
                    />
                  </label>
                </>
              )}
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
                {busy ? "Saving…" : "Save"}
              </button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
