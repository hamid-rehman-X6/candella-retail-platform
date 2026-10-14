import { apiFetch, apiGet, apiPost } from "@/lib/api";

export type StoreLocation = {
  id: string;
  name: string;
  isDefault: boolean;
  accessible: boolean;
};

export const listStores = (tenantId: string) =>
  apiGet<{ stores: StoreLocation[] }>("/stores", tenantId);

export const createStore = (tenantId: string, name: string) =>
  apiPost<{ store: StoreLocation }>("/stores", { name }, tenantId);

export const renameStore = (tenantId: string, storeId: string, name: string) =>
  apiFetch<{ updated: boolean }>(`/stores/${encodeURIComponent(storeId)}`, {
    method: "PATCH",
    body: JSON.stringify({ name }),
    tenantId,
  });

export const setDefaultStore = (tenantId: string, storeId: string) =>
  apiPost<{ updated: boolean }>(
    `/stores/${encodeURIComponent(storeId)}/default`,
    {},
    tenantId,
  );

export const listStoreAccess = (tenantId: string, storeId: string) =>
  apiGet<{ userIds: string[] }>(
    `/stores/${encodeURIComponent(storeId)}/access`,
    tenantId,
  );

export const setStoreAccess = (
  tenantId: string,
  storeId: string,
  userId: string,
  allowed: boolean,
) =>
  apiPost<{ updated: boolean }>(
    `/stores/${encodeURIComponent(storeId)}/access`,
    { userId, allowed },
    tenantId,
  );
