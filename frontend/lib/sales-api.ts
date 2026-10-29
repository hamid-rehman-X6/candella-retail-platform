import { apiGet, apiPost } from "@/lib/api";

export type SaleItem = {
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  quantity: number;
  unitPriceMinor: number;
  lineTotalMinor: number;
  /** Units already taken back; quantity - returnedQuantity can still be returned. */
  returnedQuantity: number;
  /** The tax as charged on this line; null name means not taxed. */
  taxName: string | null;
  taxRateBps: number | null;
  taxInclusive: boolean | null;
  taxMinor: number;
  /** Item promotion plus this line's share of a bill discount. */
  discountMinor: number;
  promotionName: string | null;
};

export type ReturnSummary = {
  id: string;
  returnNumber: string;
  refundMinor: number;
  processorName: string;
  reason: string;
  createdAt: string;
};

export type Sale = {
  id: string;
  customerId: string;
  customerName: string;
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
  discountMinor: number;
  /** The bill discount as given; null when the sale had none. */
  discountKind: "promotion" | "manual" | null;
  discountName: string | null;
  discountPercentBps: number | null;
  discountReason: string | null;
  taxMinor: number;
  grandTotalMinor: number;
  cashReceivedMinor: number;
  changeMinor: number;
  createdAt: string;
  /** Returns against this sale, oldest first. The sale itself never changes. */
  returns: ReturnSummary[];
  refundedMinor: number;
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
  customerId?: string;
  idempotencyKey: string;
  items: { variantId: string; quantity: number; promotionId?: string }[];
  cashReceivedMinor: number;
  /** A bill discount: an approved promotion, or a manual % with a reason. */
  discount?: { promotionId?: string; percentBps?: number; reason?: string };
};

export const checkout = (tenantId: string, input: CheckoutInput) =>
  apiPost<{ sale: Sale }>("/sales", input, tenantId);

export const getSale = (tenantId: string, orderId: string) =>
  apiGet<{ sale: Sale }>(`/sales/${encodeURIComponent(orderId)}`, tenantId);

export type ReturnItem = {
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  quantity: number;
  unitPriceMinor: number;
  lineRefundMinor: number;
  taxMinor: number;
};

export type SaleReturn = {
  id: string;
  returnNumber: string;
  orderId: string;
  orderNumber: string;
  storeId: string;
  storeName: string;
  registerName: string;
  shiftId: string;
  reason: string;
  refundMethod: "cash";
  refundMinor: number;
  taxMinor: number;
  currencyCode: string;
  processedBy: string;
  processorName: string;
  items: ReturnItem[];
  createdAt: string;
};

export type ReturnInput = {
  storeId: string;
  registerId: string;
  idempotencyKey: string;
  reason: string;
  items: { variantId: string; quantity: number }[];
};

export const createReturn = (
  tenantId: string,
  orderId: string,
  input: ReturnInput,
) =>
  apiPost<{ return: SaleReturn }>(
    `/sales/${encodeURIComponent(orderId)}/returns`,
    input,
    tenantId,
  );

export const getReturn = (tenantId: string, returnId: string) =>
  apiGet<{ return: SaleReturn }>(
    `/returns/${encodeURIComponent(returnId)}`,
    tenantId,
  );
