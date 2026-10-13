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
  orderNumber: string;
  status: "completed";
  currencyCode: string;
  items: SaleItem[];
  subtotalMinor: number;
  grandTotalMinor: number;
  cashReceivedMinor: number;
  changeMinor: number;
  createdAt: string;
};

export type CheckoutInput = {
  idempotencyKey: string;
  items: { variantId: string; quantity: number }[];
  cashReceivedMinor: number;
};

export const checkout = (tenantId: string, input: CheckoutInput) =>
  apiPost<{ sale: Sale }>("/sales", input, tenantId);

export const getSale = (tenantId: string, orderId: string) =>
  apiGet<{ sale: Sale }>(`/sales/${encodeURIComponent(orderId)}`, tenantId);
