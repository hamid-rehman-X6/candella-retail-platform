import { apiGet, apiPost } from "@/lib/api";

export type PurchaseItem = {
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  quantity: number;
  receivedQuantity: number;
  unitCostMinor: number;
  lineCostMinor: number;
};
export type ReceiptItem = {
  variantId: string;
  quantity: number;
  unitCostMinor: number;
  lineCostMinor: number;
};
export type GoodsReceipt = {
  id: string;
  purchaseOrderId: string;
  totalCostMinor: number;
  createdAt: string;
  items: ReceiptItem[];
};
export type PurchaseOrder = {
  id: string;
  number: string;
  storeId: string;
  storeName: string;
  supplierId: string;
  supplierName: string;
  status: "draft" | "approved" | "partially_received" | "received";
  currencyCode: string;
  totalCostMinor: number;
  createdAt: string;
  approvedAt: string | null;
  items: PurchaseItem[];
  receipts: GoodsReceipt[];
};
export type PurchaseSummary = Pick<
  PurchaseOrder,
  | "id"
  | "number"
  | "storeId"
  | "supplierName"
  | "status"
  | "currencyCode"
  | "totalCostMinor"
  | "createdAt"
>;
export type CreatePurchaseInput = {
  storeId: string;
  supplierId: string;
  idempotencyKey: string;
  items: { variantId: string; quantity: number; unitCostMinor: number }[];
};
export type ReceivePurchaseInput = {
  idempotencyKey: string;
  items: { variantId: string; quantity: number }[];
};

export const listPurchases = (tenantId: string, storeId: string, offset = 0) =>
  apiGet<{ orders: PurchaseSummary[] }>(
    `/purchase-orders?storeId=${encodeURIComponent(storeId)}&limit=50&offset=${offset}`,
    tenantId,
  );
export const getPurchase = (tenantId: string, orderId: string) =>
  apiGet<{ order: PurchaseOrder }>(
    `/purchase-orders/${encodeURIComponent(orderId)}`,
    tenantId,
  );
export const createPurchase = (tenantId: string, input: CreatePurchaseInput) =>
  apiPost<{ order: PurchaseOrder }>("/purchase-orders", input, tenantId);
export const approvePurchase = (tenantId: string, orderId: string) =>
  apiPost<{ order: PurchaseOrder }>(
    `/purchase-orders/${encodeURIComponent(orderId)}/approve`,
    undefined,
    tenantId,
  );
export const receivePurchase = (
  tenantId: string,
  orderId: string,
  input: ReceivePurchaseInput,
) =>
  apiPost<{ order: PurchaseOrder; receipt: GoodsReceipt }>(
    `/purchase-orders/${encodeURIComponent(orderId)}/receipts`,
    input,
    tenantId,
  );
