import { apiGet, apiPost } from "@/lib/api";

export type SaleItem = {
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  quantity: number;
  unitPriceMinor: number;
  lineTotalMinor: number;
};

export type Sale = {
  id: string;
  storeId: string;
  storeName: string;
  /** Empty for sales made before shifts existed. */
  registerName: string;
  shiftId: string;
  orderNumber: string;
  cashierId: string;
  cashierName: string;
  status: "completed";
  currencyCode: string;
  items: SaleItem[];
  subtotalMinor: number;
  grandTotalMinor: number;
  cashReceivedMinor: number;
  changeMinor: number;
  createdAt: string;
};

export type SaleSummary = Pick<
  Sale,
  | "id"
  | "storeId"
  | "storeName"
  | "orderNumber"
  | "cashierId"
  | "cashierName"
  | "currencyCode"
  | "grandTotalMinor"
  | "createdAt"
>;

export type SalesFilters = {
  fromDate?: string;
  toDate?: string;
  orderNumber?: string;
  cashier?: string;
  minAmountMinor?: number;
  maxAmountMinor?: number;
  limit?: number;
  offset?: number;
};

export const listSales = (tenantId: string, filters: SalesFilters) => {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") query.set(key, String(value));
  }
  return apiGet<{ sales: SaleSummary[]; hasMore: boolean }>(
    `/sales?${query.toString()}`,
    tenantId,
  );
};

export type CheckoutInput = {
  storeId: string;
  registerId: string;
  idempotencyKey: string;
  items: { variantId: string; quantity: number }[];
  cashReceivedMinor: number;
};

export const checkout = (tenantId: string, input: CheckoutInput) =>
  apiPost<{ sale: Sale }>("/sales", input, tenantId);

export const getSale = (tenantId: string, orderId: string) =>
  apiGet<{ sale: Sale }>(`/sales/${encodeURIComponent(orderId)}`, tenantId);
