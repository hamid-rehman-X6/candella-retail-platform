import { apiGet, apiPost } from "@/lib/api";

export type ShiftSummary = {
  id: string;
  openedBy: string;
  openerName: string;
  openedAt: string;
};

export type Register = {
  id: string;
  storeId: string;
  name: string;
  openSession: ShiftSummary | null;
};

export type Shift = {
  id: string;
  storeId: string;
  storeName: string;
  registerId: string;
  registerName: string;
  status: "open" | "closed";
  currencyCode: string;
  openedBy: string;
  openerName: string;
  openedAt: string;
  openingCashMinor: number;
  closedBy: string | null;
  closerName: string | null;
  closedAt: string | null;
  /** Live while the shift is open; fixed at closing. */
  salesCount: number;
  cashSalesMinor: number;
  expectedCashMinor: number;
  countedCashMinor: number | null;
  varianceMinor: number | null;
  varianceReason: string | null;
};

export const listRegisters = (tenantId: string, storeId: string) =>
  apiGet<{ registers: Register[] }>(
    `/registers?storeId=${encodeURIComponent(storeId)}`,
    tenantId,
  );

export const createRegister = (
  tenantId: string,
  storeId: string,
  name: string,
) => apiPost<{ register: Register }>("/registers", { storeId, name }, tenantId);

export const openShift = (
  tenantId: string,
  input: { storeId: string; registerId: string; openingCashMinor: number },
) => apiPost<{ shift: Shift }>("/shifts", input, tenantId);

export const getShift = (tenantId: string, shiftId: string) =>
  apiGet<{ shift: Shift }>(`/shifts/${encodeURIComponent(shiftId)}`, tenantId);

export const closeShift = (
  tenantId: string,
  shiftId: string,
  input: {
    countedCashMinor: number;
    expectedCashMinor: number;
    varianceReason: string;
  },
) =>
  apiPost<{ shift: Shift }>(
    `/shifts/${encodeURIComponent(shiftId)}/close`,
    input,
    tenantId,
  );

export const listShifts = (
  tenantId: string,
  filters: {
    storeId: string;
    status?: "open" | "closed" | "";
    limit?: number;
    offset?: number;
  },
) => {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") query.set(key, String(value));
  }
  return apiGet<{ shifts: Shift[]; hasMore: boolean }>(
    `/shifts?${query.toString()}`,
    tenantId,
  );
};

/**
 * The counter is a property of the till, not the person: whoever rings up a
 * sale on this device puts the cash in this counter's drawer. So it is
 * remembered per device (localStorage), per workspace and store.
 */
const counterKey = (tenantId: string, storeId: string) =>
  `candella.register.${tenantId}.${storeId}`;

export function rememberedCounter(tenantId: string, storeId: string) {
  try {
    return localStorage.getItem(counterKey(tenantId, storeId)) ?? "";
  } catch {
    return "";
  }
}

export function rememberCounter(
  tenantId: string,
  storeId: string,
  registerId: string,
) {
  try {
    localStorage.setItem(counterKey(tenantId, storeId), registerId);
  } catch {
    // Private mode or blocked storage: the cashier just picks again next time.
  }
}
