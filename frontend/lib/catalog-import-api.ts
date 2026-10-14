import type { ApiResponse } from "@candella/types";
import { API_BASE_URL, ApiError, TENANT_HEADER } from "@/lib/api";

export type ImportRowResult = {
  line: number;
  productName: string;
  sku: string;
  action: "create" | "update" | "invalid";
  errors: string[];
};

export type ImportPreview = {
  rows: ImportRowResult[];
  createCount: number;
  updateCount: number;
  invalidCount: number;
  hash: string;
};

export type ImportResult = {
  created: number;
  updated: number;
  replayed: boolean;
  rows: ImportRowResult[];
};

async function upload<T>(
  path: string,
  tenantId: string,
  form: FormData,
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}/api/v1${path}`, {
      method: "POST",
      body: form,
      credentials: "include",
      headers: { [TENANT_HEADER]: tenantId },
    });
  } catch {
    throw new ApiError("network_error", "Can't reach the server. Try again.");
  }
  const payload = (await response.json()) as ApiResponse<T>;
  if (!payload.success)
    throw new ApiError(payload.error.code, payload.error.message);
  return payload.data;
}

function uploadForm(file: File, mode: "create" | "update", storeId: string) {
  const form = new FormData();
  form.set("file", file);
  form.set("mode", mode);
  form.set("storeId", storeId);
  return form;
}

export const previewProductImport = (
  tenantId: string,
  file: File,
  mode: "create" | "update",
  storeId: string,
) =>
  upload<{ preview: ImportPreview }>(
    "/products/import/preview",
    tenantId,
    uploadForm(file, mode, storeId),
  );

export const confirmProductImport = (
  tenantId: string,
  file: File,
  mode: "create" | "update",
  storeId: string,
  previewHash: string,
  idempotencyKey: string,
) => {
  const form = uploadForm(file, mode, storeId);
  form.set("previewHash", previewHash);
  form.set("idempotencyKey", idempotencyKey);
  return upload<{ result: ImportResult }>(
    "/products/import/confirm",
    tenantId,
    form,
  );
};

export async function downloadProductCSV(
  tenantId: string,
  kind: "template" | "export",
  storeId: string,
) {
  const path =
    kind === "template"
      ? "/products/import/template.csv"
      : `/products/export.csv?storeId=${encodeURIComponent(storeId)}`;
  const response = await fetch(`${API_BASE_URL}/api/v1${path}`, {
    credentials: "include",
    headers: { [TENANT_HEADER]: tenantId },
  });
  if (!response.ok) {
    const payload = (await response.json()) as ApiResponse<never>;
    if (!payload.success)
      throw new ApiError(payload.error.code, payload.error.message);
    throw new Error("Could not download the CSV file.");
  }
  const blob = await response.blob();
  const href = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = href;
  link.download =
    kind === "template"
      ? "candella-products-template.csv"
      : "candella-products.csv";
  link.click();
  URL.revokeObjectURL(href);
}
