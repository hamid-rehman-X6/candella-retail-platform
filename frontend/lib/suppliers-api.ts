import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type Supplier = {
  id: string;
  name: string;
  phone: string;
  email: string;
  address: string;
  status: "active" | "archived";
  createdAt: string;
  updatedAt: string;
};

export type SupplierInput = Pick<
  Supplier,
  "name" | "phone" | "email" | "address"
>;

export type SupplierSource = {
  supplierId: string;
  variantId: string;
  productName: string;
  variantName: string;
  sku: string;
  productStatus: "active" | "archived";
  supplierSku: string;
  costMinor: number | null;
  createdAt: string;
  updatedAt: string;
};

export const listSuppliers = (tenantId: string, search = "", offset = 0) => {
  const query = new URLSearchParams({ limit: "100", offset: String(offset) });
  if (search.trim()) query.set("search", search.trim());
  return apiGet<{ suppliers: Supplier[] }>(`/suppliers?${query}`, tenantId);
};

export const createSupplier = (tenantId: string, input: SupplierInput) =>
  apiPost<{ supplier: Supplier }>("/suppliers", input, tenantId);

export const updateSupplier = (
  tenantId: string,
  supplierId: string,
  input: Partial<SupplierInput> & { status?: Supplier["status"] },
) =>
  apiFetch<{ supplier: Supplier }>(`/suppliers/${supplierId}`, {
    method: "PATCH",
    body: JSON.stringify(input),
    tenantId,
  });

export const listSupplierSources = (
  tenantId: string,
  supplierId: string,
  offset = 0,
) =>
  apiGet<{ sources: SupplierSource[] }>(
    `/suppliers/${supplierId}/sources?limit=100&offset=${offset}`,
    tenantId,
  );

export const saveSupplierSource = (
  tenantId: string,
  supplierId: string,
  input: { variantId: string; supplierSku: string; costMinor: number | null },
) =>
  apiPost<{ source: SupplierSource }>(
    `/suppliers/${supplierId}/sources`,
    input,
    tenantId,
  );
