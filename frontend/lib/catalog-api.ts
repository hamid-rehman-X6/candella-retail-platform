import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type Variant = {
  id: string;
  productId: string;
  name: string;
  sku: string;
  barcode: string | null;
  priceMinor: number;
  quantity: number;
};

export type Product = {
  id: string;
  name: string;
  description: string;
  status: "active" | "archived";
  /** The rate charged on every variant; null means not taxed. */
  taxRateId: string | null;
  taxRateName: string | null;
  variants: Variant[];
  createdAt: string;
};

export type StockItem = {
  variantId: string;
  productId: string;
  productName: string;
  variantName: string;
  sku: string;
  barcode: string | null;
  priceMinor: number;
  quantity: number;
  reorderThreshold: number | null;
  preferredSupplierId: string | null;
  preferredSupplierName: string | null;
  configuredSupplierId: string | null;
  taxName: string | null;
  taxRateBps: number | null;
  taxInclusive: boolean | null;
};

export type ReorderItem = StockItem & { suggestedQuantity: number };

export type CatalogSummary = {
  productCount: number;
  variantCount: number;
  unitsOnHand: number;
  outOfStock: number;
};

export type Movement = {
  id: string;
  storeId: string;
  orderId: string | null;
  purchaseOrderId: string | null;
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  kind: "receive" | "count" | "sale" | "return" | "purchase";
  delta: number;
  quantityBefore: number;
  quantityAfter: number;
  note: string;
  createdAt: string;
};

export type VariantInput = {
  name: string;
  sku: string;
  barcode: string;
  priceMinor: number;
};

function query(search: string, offset = 0) {
  const params = new URLSearchParams({ limit: "100", offset: String(offset) });
  if (search.trim()) params.set("search", search.trim());
  return params;
}

export const listProducts = (
  tenantId: string,
  storeId: string,
  search = "",
  offset = 0,
) =>
  apiGet<{ products: Product[] }>(
    `/products?${query(search, offset)}&storeId=${encodeURIComponent(storeId)}`,
    tenantId,
  );

export const getCatalogSummary = (tenantId: string, storeId: string) =>
  apiGet<{ summary: CatalogSummary }>(
    `/catalog/summary?storeId=${encodeURIComponent(storeId)}`,
    tenantId,
  );

export const createProduct = (
  tenantId: string,
  input: {
    storeId?: string;
    name: string;
    description: string;
    variant: VariantInput;
    initialQuantity?: number;
    taxRateId?: string;
  },
) => apiPost<{ product: Product }>("/products", input, tenantId);

export const updateProduct = (
  tenantId: string,
  productId: string,
  input: Partial<Pick<Product, "name" | "description" | "status">> & {
    /** "" removes the product's tax. */
    taxRateId?: string;
  },
) =>
  apiFetch<{ product: Product }>(`/products/${productId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
    tenantId,
  });

export const addVariant = (
  tenantId: string,
  productId: string,
  input: VariantInput,
) =>
  apiPost<{ variant: Variant }>(
    `/products/${productId}/variants`,
    input,
    tenantId,
  );

export const updateVariant = (
  tenantId: string,
  variantId: string,
  input: VariantInput,
) =>
  apiFetch<{ variant: Variant }>(`/products/variants/${variantId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
    tenantId,
  });

export const listStock = (
  tenantId: string,
  storeId: string,
  search = "",
  offset = 0,
) =>
  apiGet<{ items: StockItem[]; storeId: string }>(
    `/inventory?${query(search, offset)}&storeId=${encodeURIComponent(storeId)}`,
    tenantId,
  );

export const listLowStock = (tenantId: string, storeId: string, offset = 0) =>
  apiGet<{ items: ReorderItem[] }>(
    `/inventory/low-stock?storeId=${encodeURIComponent(storeId)}&limit=100&offset=${offset}`,
    tenantId,
  );

export const saveReorder = (
  tenantId: string,
  variantId: string,
  input: {
    storeId: string;
    threshold: number | null;
    preferredSupplierId: string | null;
  },
) =>
  apiFetch<{ saved: boolean }>(
    `/inventory/reorder/${encodeURIComponent(variantId)}`,
    {
      method: "PUT",
      body: JSON.stringify(input),
      tenantId,
    },
  );

export const listMovements = (tenantId: string, storeId: string, offset = 0) =>
  apiGet<{ movements: Movement[] }>(
    `/inventory/movements?limit=50&offset=${offset}&storeId=${encodeURIComponent(storeId)}`,
    tenantId,
  );

export const recordMovement = (
  tenantId: string,
  input:
    | {
        idempotencyKey: string;
        storeId: string;
        variantId: string;
        kind: "receive";
        quantity: number;
        note: string;
      }
    | {
        idempotencyKey: string;
        storeId: string;
        variantId: string;
        kind: "count";
        countedQuantity: number;
        note: string;
      },
) => apiPost<{ movement: Movement }>("/inventory/movements", input, tenantId);
