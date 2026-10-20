import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type Customer = {
  id: string;
  name: string;
  phone: string;
  email: string;
  status: "active" | "archived";
  createdAt: string;
  updatedAt: string;
};
export type CustomerInput = Pick<Customer, "name" | "phone" | "email">;
export type CustomerSale = {
  id: string;
  orderNumber: string;
  storeName: string;
  currencyCode: string;
  grandTotalMinor: number;
  createdAt: string;
};

export function listCustomers(tenantId: string, search = "", offset = 0) {
  const params = new URLSearchParams({ limit: "50", offset: String(offset) });
  if (search.trim()) params.set("search", search.trim());
  return apiGet<{ customers: Customer[] }>(`/customers?${params}`, tenantId);
}
export const createCustomer = (tenantId: string, input: CustomerInput) =>
  apiPost<{ customer: Customer }>("/customers", input, tenantId);
export const updateCustomer = (
  tenantId: string,
  customerId: string,
  input: Partial<CustomerInput> & { status?: Customer["status"] },
) =>
  apiFetch<{ customer: Customer }>(
    `/customers/${encodeURIComponent(customerId)}`,
    {
      method: "PATCH",
      body: JSON.stringify(input),
      tenantId,
    },
  );
export const customerSales = (
  tenantId: string,
  customerId: string,
  offset = 0,
) =>
  apiGet<{ sales: CustomerSale[] }>(
    `/customers/${encodeURIComponent(customerId)}/sales?limit=50&offset=${offset}`,
    tenantId,
  );
