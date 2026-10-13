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
};

export type CatalogSummary = {
  productCount: number;
  variantCount: number;
  unitsOnHand: number;
  outOfStock: number;
};

export type Movement = {
  id: string;
  orderId: string | null;
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  kind: "receive" | "count" | "sale";
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

export const listProducts = (tenantId: string, search = "", offset = 0) =>
  apiGet<{ products: Product[] }>(
    `/products?${query(search, offset)}`,
    tenantId,
  );

export const getCatalogSummary = (tenantId: string) =>
  apiGet<{ summary: CatalogSummary }>("/catalog/summary", tenantId);

export const createProduct = (
  tenantId: string,
  input: { name: string; description: string; variant: VariantInput },
) => apiPost<{ product: Product }>("/products", input, tenantId);

export const updateProduct = (
  tenantId: string,
  productId: string,
  input: Partial<Pick<Product, "name" | "description" | "status">>,
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

export const listStock = (tenantId: string, search = "", offset = 0) =>
  apiGet<{ items: StockItem[]; storeId: string }>(
    `/inventory?${query(search, offset)}`,
    tenantId,
  );

export const listMovements = (tenantId: string, offset = 0) =>
  apiGet<{ movements: Movement[] }>(
    `/inventory/movements?limit=50&offset=${offset}`,
    tenantId,
  );

export const recordMovement = (
  tenantId: string,
  input:
    | { variantId: string; kind: "receive"; quantity: number; note: string }
    | {
        variantId: string;
        kind: "count";
        countedQuantity: number;
        note: string;
      },
) => apiPost<{ movement: Movement }>("/inventory/movements", input, tenantId);

export function formatMoney(minor: number, currency: string) {
  const formatter = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
  });
  const digits = formatter.resolvedOptions().maximumFractionDigits ?? 2;
  return formatter.format(minor / 10 ** digits);
}

export function minorDigits(currency: string) {
  return (
    new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
    }).resolvedOptions().maximumFractionDigits ?? 2
  );
}

export function parseMoney(input: string, currency: string): number | null {
  const digits = minorDigits(currency);
  const pattern = new RegExp(
    digits === 0 ? "^\\d+$" : `^\\d+(?:\\.\\d{1,${digits}})?$`,
  );
  if (!pattern.test(input.trim())) return null;
  const [whole, decimal = ""] = input.trim().split(".");
  const value =
    Number(whole) * 10 ** digits + Number(decimal.padEnd(digits, "0"));
  return Number.isSafeInteger(value) && value <= 1_000_000_000_000
    ? value
    : null;
}
