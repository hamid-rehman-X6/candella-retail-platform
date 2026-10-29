import { apiGet, apiPost, apiFetch } from "@/lib/api";

export type Promotion = {
  id: string;
  name: string;
  kind: "percent" | "amount";
  percentBps: number | null;
  /** Off the whole order, or off each unit for an item promotion. */
  amountMinor: number | null;
  appliesTo: "order" | "item";
  minSubtotalMinor: number;
  /** Inclusive business days, YYYY-MM-DD; null is open-ended. */
  startsOn: string | null;
  endsOn: string | null;
  status: "active" | "archived";
  createdAt: string;
};

export type PromotionInput = Omit<Promotion, "id" | "status" | "createdAt">;

export const listPromotions = (tenantId: string, includeArchived = false) =>
  apiGet<{ promotions: Promotion[] }>(
    `/promotions${includeArchived ? "?include=archived" : ""}`,
    tenantId,
  );

export const createPromotion = (tenantId: string, input: PromotionInput) =>
  apiPost<{ promotion: Promotion }>("/promotions", input, tenantId);

export const updatePromotion = (
  tenantId: string,
  promotionId: string,
  input: PromotionInput & { status?: "active" | "archived" },
) =>
  apiFetch<{ promotion: Promotion }>(
    `/promotions/${encodeURIComponent(promotionId)}`,
    { method: "PUT", body: JSON.stringify(input), tenantId },
  );

export const getDiscountSettings = (tenantId: string) =>
  apiGet<{ settings: { cashierLimitBps: number } }>(
    "/discount-settings",
    tenantId,
  );

export const saveDiscountSettings = (
  tenantId: string,
  cashierLimitBps: number,
) =>
  apiFetch<{ settings: { cashierLimitBps: number } }>("/discount-settings", {
    method: "PUT",
    body: JSON.stringify({ cashierLimitBps }),
    tenantId,
  });

/** Whether a promotion can be used today (YYYY-MM-DD in the workspace zone). */
export function usableOn(p: Promotion, today: string): boolean {
  return (
    p.status === "active" &&
    (!p.startsOn || p.startsOn <= today) &&
    (!p.endsOn || p.endsOn >= today)
  );
}
