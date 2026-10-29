import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type TaxRate = {
  id: string;
  name: string;
  /** Basis points: 1800 = 18%. */
  rateBps: number;
  /** True when shelf prices already contain the tax. */
  inclusive: boolean;
  status: "active" | "archived";
  productCount: number;
  createdAt: string;
};

export const listTaxRates = (tenantId: string, includeArchived = false) =>
  apiGet<{ taxRates: TaxRate[] }>(
    `/tax-rates${includeArchived ? "?include=archived" : ""}`,
    tenantId,
  );

export const createTaxRate = (
  tenantId: string,
  input: { name: string; rateBps: number; inclusive: boolean },
) => apiPost<{ taxRate: TaxRate }>("/tax-rates", input, tenantId);

export const updateTaxRate = (
  tenantId: string,
  rateId: string,
  input: Partial<{
    name: string;
    rateBps: number;
    inclusive: boolean;
    status: "active" | "archived";
  }>,
) =>
  apiFetch<{ taxRate: TaxRate }>(`/tax-rates/${encodeURIComponent(rateId)}`, {
    method: "PATCH",
    body: JSON.stringify(input),
    tenantId,
  });
