"use client";

import Link from "next/link";
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { FileUp, PackagePlus, Plus, Search } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { useStoreSelection } from "@/lib/use-store-selection";
import { Select } from "@/components/ui/select";
import { ConfirmDialog, Modal, modalButton } from "@/components/ui/modal";
import { toast } from "@/lib/toast";
import {
  addVariant,
  createProduct,
  listProducts,
  updateProduct,
  updateVariant,
  type Product,
  type Variant,
} from "@/lib/catalog-api";
import { formatMoney, minorDigits, parseMoney } from "@/lib/money";

const field =
  "w-full rounded-xl border border-line bg-surface px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent";
const button =
  "inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black hover:bg-accent-bright disabled:opacity-50";

export default function ProductsPage() {
  const { current } = useApp();
  const {
    stores,
    storeId,
    select: selectStore,
    error: storeError,
  } = useStoreSelection(current.id);
  const editable = current.role === "owner" || current.role === "admin";
  const [products, setProducts] = useState<Product[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [statusTarget, setStatusTarget] = useState<Product | null>(null);
  const [statusBusy, setStatusBusy] = useState(false);
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
  const [initialQuantity, setInitialQuantity] = useState("0");
  const formId = useId();
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const loadID = useRef(0);

  const refresh = useCallback(
    async (q = search) => {
      if (!storeId) return;
      const requestID = ++loadID.current;
      setLoading(true);
      setError("");
      try {
        const { products: next } = await listProducts(current.id, storeId, q);
        if (requestID !== loadID.current) return;
        setProducts(next);
        setOffset(0);
        setHasMore(next.length === 100);
      } catch (e) {
        if (requestID !== loadID.current) return;
        setError(e instanceof Error ? e.message : "Could not load products.");
      } finally {
        if (requestID === loadID.current) setLoading(false);
      }
    },
    [current.id, storeId, search],
  );

  useEffect(() => {
    const timer = setTimeout(() => {
      void refresh(search);
    }, 250);
    return () => clearTimeout(timer);
  }, [refresh, search]);

  function changeStore(id: string) {
    loadID.current++;
    setProducts([]);
    setLoading(true);
    selectStore(id);
  }

  function openCreate() {
    setSelected(null);
    setName("");
    setDescription("");
    setVariantName("");
    setSku("");
    setBarcode("");
    setPrice("");
    setInitialQuantity("0");
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
    const openingStock =
      initialQuantity.trim() === "" ? 0 : Number(initialQuantity);
    if (
      form === "create" &&
      (!Number.isInteger(openingStock) ||
        openingStock < 0 ||
        openingStock > 1_000_000_000)
    ) {
      setError("Enter a whole opening quantity between 0 and 1,000,000,000.");
      return;
    }
    if (form === "create" && openingStock > 0 && !storeId) {
      setError("Select the store receiving the opening stock.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (form === "create") {
        await createProduct(current.id, {
          name,
          description,
          initialQuantity: openingStock,
          storeId,
          variant: {
            name: variantName || "Standard",
            sku: sku.trim(),
            barcode,
            priceMinor,
          },
        });
      } else if (form === "variant" && selected) {
        await addVariant(current.id, selected.id, {
          name: variantName || "Standard",
          sku: sku.trim(),
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
  async function confirmStatusChange() {
    if (!statusTarget || statusBusy) return;
    const nextStatus = statusTarget.status === "active" ? "archived" : "active";
    setStatusBusy(true);
    try {
      const updated = await updateProduct(current.id, statusTarget.id, {
        status: nextStatus,
      });
      await refresh();
      setStatusTarget(null);
      toast.success(
        `${updated.product.name} ${nextStatus === "archived" ? "archived" : "restored"}`,
      );
    } catch (e) {
      toast.error(
        `Couldn't ${nextStatus === "archived" ? "archive" : "restore"} product`,
        {
          description: e instanceof Error ? e.message : "Please try again.",
        },
      );
    } finally {
      setStatusBusy(false);
    }
  }
  async function loadMore() {
    const nextOffset = offset + 100;
    const requestID = loadID.current;
    try {
      const { products: next } = await listProducts(
        current.id,
        storeId,
        search,
        nextOffset,
      );
      if (requestID !== loadID.current) return;
      setProducts((previous) => [...previous, ...next]);
      setOffset(nextOffset);
      setHasMore(next.length === 100);
    } catch (e) {
      if (requestID !== loadID.current) return;
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
            Your sellable catalog. A unique SKU is generated for each new
            variant.
          </p>
        </div>
        {editable && (
          <div className="flex flex-wrap gap-2">
            <Link
              href={workspacePath(current.slug, "/products/import")}
              className="inline-flex items-center justify-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm font-semibold hover:bg-surface"
            >
              <FileUp size={17} /> Import CSV
            </Link>
            <button type="button" onClick={openCreate} className={button}>
              <PackagePlus size={17} /> Add product
            </button>
          </div>
        )}
      </div>
      <label className="block max-w-xs text-sm font-medium">
        Stock shown for
        <Select
          ariaLabel="Product stock store"
          className={`${field} mt-2`}
          value={storeId}
          onValueChange={changeStore}
          options={[
            ...(!storeId ? [{ value: "", label: "Select a store" }] : []),
            ...stores
              .filter((location) => location.accessible)
              .map((location) => ({
                value: location.id,
                label: location.name,
              })),
          ]}
        />
      </label>
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
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}
      {storeError && (
        <p role="alert" className="text-sm text-danger-ink">
          {storeError}
        </p>
      )}
      {loading ? (
        <p className="text-sm text-muted">Loading products…</p>
      ) : products.length === 0 ? (
        <div className="rounded-2xl border border-line bg-surface p-10 text-center">
          <PackagePlus className="mx-auto text-accent-ink" size={28} />
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
                      <span className="rounded-full bg-foreground/10 px-2 py-0.5 text-xs text-muted">
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
                      onClick={() => setStatusTarget(product)}
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
                        variant.quantity === 0
                          ? "text-warning-ink"
                          : "text-muted"
                      }
                    >
                      {variant.quantity} in stock
                      {editable && product.status === "active" && (
                        <button
                          type="button"
                          onClick={() => openVariantEdit(product, variant)}
                          className="ml-3 text-xs text-accent-ink hover:underline"
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
      <ConfirmDialog
        open={!!statusTarget}
        title={
          statusTarget
            ? `${statusTarget.status === "active" ? "Archive" : "Restore"} ${statusTarget.name}?`
            : ""
        }
        message={
          statusTarget?.status === "active"
            ? "This product will no longer be available for new sales or stock receipts. Its history stays available."
            : "This product will be available for new sales and stock receipts again."
        }
        confirmLabel={
          statusTarget?.status === "active"
            ? "Archive product"
            : "Restore product"
        }
        tone={statusTarget?.status === "active" ? "danger" : "primary"}
        busy={statusBusy}
        onCancel={() => setStatusTarget(null)}
        onConfirm={() => void confirmStatusChange()}
      />
      <Modal
        open={!!form}
        onClose={() => setForm(null)}
        busy={busy}
        size="lg"
        title={
          form === "create"
            ? "Add product"
            : form === "variant"
              ? "Add variant"
              : form === "variant_edit"
                ? "Edit variant"
                : "Edit product"
        }
        description={
          form === "create"
            ? "Only the name and selling price are required."
            : form === "variant"
              ? `A new size, colour or pack of ${selected?.name ?? "this product"}.`
              : form === "variant_edit"
                ? `${selected?.name ?? ""} · ${selectedVariant?.name ?? ""}`
                : selected?.name
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
              {busy
                ? "Saving…"
                : form === "create"
                  ? "Add product"
                  : form === "variant"
                    ? "Add variant"
                    : "Save changes"}
            </button>
          </>
        }
      >
        <form
          id={formId}
          onSubmit={(e) => void submit(e)}
          className="space-y-4"
        >
          {(form === "create" || form === "edit") && (
            <>
              <label className="block text-sm font-medium">
                Product name
                <input
                  className={`${field} mt-1.5`}
                  required
                  maxLength={160}
                  placeholder="e.g. Cotton Shirt"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <label className="block text-sm font-medium">
                Description{" "}
                <span className="font-normal text-muted">(optional)</span>
                <textarea
                  className={`${field} mt-1.5 resize-y`}
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
              <label className="block text-sm font-medium">
                Variant{" "}
                <span className="font-normal text-muted">(optional)</span>
                <input
                  className={`${field} mt-1.5`}
                  maxLength={100}
                  placeholder="e.g. Large, Blue, 500ml"
                  value={variantName}
                  onChange={(e) => setVariantName(e.target.value)}
                />
                <span className="mt-1 block text-xs font-normal text-muted">
                  Leave blank if the product comes in one kind only.
                </span>
              </label>
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block text-sm font-medium">
                  SKU{" "}
                  {form !== "variant_edit" && (
                    <span className="font-normal text-muted">(optional)</span>
                  )}
                  <input
                    className={`${field} mt-1.5 font-mono uppercase placeholder:font-sans placeholder:normal-case`}
                    required={form === "variant_edit"}
                    maxLength={80}
                    placeholder={
                      form === "variant_edit" ? "" : "Created automatically"
                    }
                    value={sku}
                    onChange={(e) => setSku(e.target.value)}
                  />
                </label>
                <label className="block text-sm font-medium">
                  Barcode{" "}
                  <span className="font-normal text-muted">(optional)</span>
                  <input
                    className={`${field} mt-1.5 font-mono`}
                    maxLength={80}
                    placeholder="Scan or type"
                    value={barcode}
                    onChange={(e) => setBarcode(e.target.value)}
                  />
                </label>
              </div>
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="block text-sm font-medium">
                  Selling price ({current.currencyCode})
                  <input
                    className={`${field} mt-1.5`}
                    inputMode="decimal"
                    required
                    placeholder="0"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                  />
                </label>
                {form === "create" && (
                  <label className="block text-sm font-medium">
                    Opening stock{" "}
                    <span className="font-normal text-muted">(optional)</span>
                    <input
                      className={`${field} mt-1.5`}
                      type="number"
                      inputMode="numeric"
                      min={0}
                      max={1_000_000_000}
                      step={1}
                      value={initialQuantity}
                      onChange={(e) => setInitialQuantity(e.target.value)}
                    />
                  </label>
                )}
              </div>
              {form === "create" && (
                <div>
                  <p className="text-sm font-medium">Opening stock goes to</p>
                  <Select
                    ariaLabel="Store receiving the opening stock"
                    className={`${field} mt-1.5`}
                    value={storeId}
                    onValueChange={changeStore}
                    options={[
                      ...(!storeId
                        ? [{ value: "", label: "Select a store" }]
                        : []),
                      ...stores
                        .filter((location) => location.accessible)
                        .map((location) => ({
                          value: location.id,
                          label: location.name,
                        })),
                    ]}
                  />
                  <p className="mt-1 text-xs text-muted">
                    You can receive more stock later from Inventory.
                  </p>
                </div>
              )}
            </>
          )}
          {error && (
            <p
              role="alert"
              className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
            >
              {error}
            </p>
          )}
        </form>
      </Modal>
    </div>
  );
}
