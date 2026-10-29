import { API_BASE_URL, TENANT_HEADER, apiGet } from "@/lib/api";

export type ReportDay = {
  date: string;
  salesCount: number;
  grossCashMinor: number;
  refundsCount: number;
  refundsMinor: number;
  netSalesMinor: number;
  /** Tax charged on the day's sales, less the tax handed back in refunds. */
  taxMinor: number;
  refundedTaxMinor: number;
  netTaxMinor: number;
};
export type TopProduct = {
  productId: string;
  productName: string;
  unitsSold: number;
  grossSalesMinor: number;
};
export type MovementTotal = {
  kind: string;
  movements: number;
  unitsIn: number;
  unitsOut: number;
  netUnits: number;
};
export type DailyReport = {
  storeId: string;
  storeName: string;
  currencyCode: string;
  timezone: string;
  fromDate: string;
  toDate: string;
  totals: ReportDay;
  days: ReportDay[];
  topProducts: TopProduct[];
  stockMovements: MovementTotal[];
};

function reportQuery(storeId: string, from: string, to: string) {
  return new URLSearchParams({ storeId, from, to }).toString();
}

export const getDailyReport = (
  tenantId: string,
  storeId: string,
  from: string,
  to: string,
) =>
  apiGet<{ report: DailyReport }>(
    `/reports/daily?${reportQuery(storeId, from, to)}`,
    tenantId,
  );

export async function downloadDailyReportCSV(
  tenantId: string,
  storeId: string,
  from: string,
  to: string,
) {
  let result: Response;
  try {
    result = await fetch(
      `${API_BASE_URL}/api/v1/reports/daily.csv?${reportQuery(storeId, from, to)}`,
      { credentials: "include", headers: { [TENANT_HEADER]: tenantId } },
    );
  } catch {
    throw new Error(
      "Can't reach the server. Check your connection and try again.",
    );
  }
  if (!result.ok) {
    const body = await result.json().catch(() => null);
    throw new Error(body?.error?.message ?? "Could not export this report.");
  }
  const blob = await result.blob();
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `candella-report-${from}-to-${to}.csv`;
  document.body.appendChild(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 0);
}
