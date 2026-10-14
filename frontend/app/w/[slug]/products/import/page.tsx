"use client";

import Link from "next/link";
import { useRef, useState } from "react";
import { ArrowLeft, Download, Upload } from "lucide-react";
import { useApp, workspacePath } from "@/components/app/app-context";
import { useStoreSelection } from "@/lib/use-store-selection";
import {
  confirmProductImport,
  downloadProductCSV,
  previewProductImport,
  type ImportPreview,
  type ImportResult,
} from "@/lib/catalog-import-api";

export default function ImportProductsPage() {
  const { current } = useApp();
  const {
    stores,
    storeId,
    selectedStore,
    select: selectStore,
    error: storeError,
  } = useStoreSelection(current.id);
  const [file, setFile] = useState<File | null>(null);
  const [mode, setMode] = useState<"create" | "update">("create");
  const [preview, setPreview] = useState<ImportPreview | null>(null);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const importKey = useRef<string | null>(null);
  const editable = current.role === "owner" || current.role === "admin";

  function reset() {
    setPreview(null);
    setResult(null);
    setError("");
    importKey.current = null;
  }

  async function inspect() {
    if (!file || !storeId) return;
    setBusy(true);
    setError("");
    setPreview(null);
    setResult(null);
    try {
      const response = await previewProductImport(
        current.id,
        file,
        mode,
        storeId,
      );
      setPreview(response.preview);
      importKey.current = crypto.randomUUID();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Could not preview this CSV file.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function confirm() {
    if (!file || !preview || preview.invalidCount > 0 || !storeId) return;
    setBusy(true);
    setError("");
    importKey.current ??= crypto.randomUUID();
    try {
      const response = await confirmProductImport(
        current.id,
        file,
        mode,
        storeId,
        preview.hash,
        importKey.current,
      );
      setResult(response.result);
      setPreview(null);
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : "Import failed. Try again.",
      );
    } finally {
      setBusy(false);
    }
  }

  async function download(kind: "template" | "export") {
    try {
      await downloadProductCSV(current.id, kind, storeId);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Download failed.");
    }
  }

  function downloadErrors() {
    if (!preview) return;
    const csv = [
      "line,product_name,sku,errors",
      ...preview.rows
        .filter((row) => row.errors.length > 0)
        .map((row) =>
          [row.line, row.productName, row.sku, row.errors.join("; ")]
            .map((value) => `"${String(value).replaceAll('"', '""')}"`)
            .join(","),
        ),
    ].join("\r\n");
    const href = URL.createObjectURL(
      new Blob([csv], { type: "text/csv;charset=utf-8" }),
    );
    const link = document.createElement("a");
    link.href = href;
    link.download = "candella-import-errors.csv";
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 0);
  }

  if (!editable)
    return (
      <p role="alert" className="text-sm text-muted">
        Only workspace owners and admins can import products.
      </p>
    );

  return (
    <div className="mx-auto max-w-5xl space-y-6 pb-10">
      <div>
        <Link
          href={workspacePath(current.slug, "/products")}
          className="inline-flex items-center gap-2 text-sm text-accent hover:underline"
        >
          <ArrowLeft className="h-4 w-4" /> Products
        </Link>
        <h1 className="mt-3 text-3xl font-semibold">Import products</h1>
        <p className="mt-1 text-sm text-muted">
          Preview the CSV before any product or stock changes are saved.
        </p>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300"
        >
          {error}
        </p>
      )}
      {storeError && (
        <p role="alert" className="text-sm text-red-300">
          {storeError}
        </p>
      )}

      <section className="space-y-4 rounded-2xl border border-line bg-surface p-5">
        <label className="block max-w-xs text-sm font-medium">
          Stock location
          <select
            aria-label="CSV stock store"
            value={storeId}
            disabled={busy}
            onChange={(event) => {
              selectStore(event.target.value);
              reset();
            }}
            className="mt-2 w-full rounded-xl border border-line bg-canvas p-2.5 text-sm"
          >
            {!storeId && <option value="">Select a store</option>}
            {stores
              .filter((location) => location.accessible)
              .map((location) => (
                <option key={location.id} value={location.id}>
                  {location.name}
                </option>
              ))}
          </select>
        </label>
        <p className="text-sm text-muted">
          New products receive opening stock at{" "}
          {selectedStore?.name ?? "the selected store"}. Export shows stock for
          that location.
        </p>
        <div className="flex flex-wrap gap-3">
          <button
            type="button"
            onClick={() => void download("template")}
            className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm"
          >
            <Download className="h-4 w-4" /> Download template
          </button>
          <button
            type="button"
            disabled={!storeId}
            onClick={() => void download("export")}
            className="inline-flex items-center gap-2 rounded-xl border border-line px-4 py-2.5 text-sm"
          >
            <Download className="h-4 w-4" /> Export catalog
          </button>
        </div>
        <p className="text-sm text-muted">
          Template columns: product_name, variant_name, sku (optional for new
          products), barcode (optional), selling_price in {current.currencyCode}
          , and opening_quantity in whole units. Rows with the same product name
          become variants of one product. Maximum 5 MB and 5,000 rows.
        </p>
        <div className="grid gap-4 sm:grid-cols-2">
          <label className="text-sm font-medium">
            CSV file
            <input
              type="file"
              accept=".csv,text/csv"
              className="mt-2 block w-full rounded-xl border border-line bg-canvas p-2.5 text-sm"
              onChange={(event) => {
                setFile(event.target.files?.[0] ?? null);
                reset();
              }}
            />
          </label>
          <label className="text-sm font-medium">
            Import mode
            <select
              value={mode}
              onChange={(event) => {
                setMode(event.target.value as "create" | "update");
                reset();
              }}
              className="mt-2 block w-full rounded-xl border border-line bg-canvas p-2.5 text-sm"
            >
              <option value="create">Create new products only</option>
              <option value="update">Update existing products by SKU</option>
            </select>
          </label>
        </div>
        <p className="text-xs text-muted">
          Create mode rejects existing SKUs and barcodes. Update mode changes
          names, barcodes and selling prices only. It never changes existing
          stock; use Inventory for stock corrections.
        </p>
        <button
          type="button"
          disabled={!file || !storeId || busy}
          onClick={() => void inspect()}
          className="inline-flex items-center gap-2 rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black disabled:opacity-50"
        >
          <Upload className="h-4 w-4" />{" "}
          {busy ? "Working..." : "Preview import"}
        </button>
      </section>

      {preview && (
        <section className="space-y-4 rounded-2xl border border-line bg-surface p-5">
          <h2 className="text-lg font-semibold">Preview</h2>
          <p className="text-sm text-muted">
            {preview.createCount} to create · {preview.updateCount} to update ·{" "}
            {preview.invalidCount} invalid
          </p>
          {preview.invalidCount > 0 && (
            <button
              type="button"
              onClick={downloadErrors}
              className="text-sm text-accent hover:underline"
            >
              Download error report
            </button>
          )}
          <div className="max-h-96 overflow-auto rounded-xl border border-line">
            <table className="w-full min-w-[580px] text-left text-sm">
              <thead className="sticky top-0 bg-canvas-raised text-muted">
                <tr>
                  <th className="p-3">Line</th>
                  <th className="p-3">Product</th>
                  <th className="p-3">SKU</th>
                  <th className="p-3">Action / error</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {preview.rows.map((row) => (
                  <tr key={row.line}>
                    <td className="p-3">{row.line}</td>
                    <td className="p-3">{row.productName || "—"}</td>
                    <td className="p-3 font-mono">{row.sku || "Auto"}</td>
                    <td className="p-3">
                      {row.errors.length ? (
                        <span className="text-red-300">
                          {row.errors.join(" ")}
                        </span>
                      ) : (
                        row.action
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <button
            type="button"
            disabled={busy || preview.invalidCount > 0}
            onClick={() => void confirm()}
            className="rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black disabled:opacity-50"
          >
            {busy
              ? "Importing..."
              : `Confirm ${preview.createCount + preview.updateCount} rows`}
          </button>
        </section>
      )}

      {result && (
        <section
          role="status"
          className="rounded-2xl border border-emerald-500/30 bg-emerald-500/10 p-5 text-sm"
        >
          <h2 className="text-lg font-semibold">Import complete</h2>
          <p className="mt-2">
            {result.created} variants created · {result.updated} variants
            updated.
          </p>
          <div className="mt-4 max-h-72 overflow-auto rounded-xl border border-line bg-surface">
            <table className="w-full min-w-[500px] text-left text-sm">
              <thead className="sticky top-0 bg-canvas-raised text-muted">
                <tr>
                  <th className="p-3">Line</th>
                  <th className="p-3">Product</th>
                  <th className="p-3">SKU</th>
                  <th className="p-3">Saved as</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-line">
                {result.rows.map((row) => (
                  <tr key={row.line}>
                    <td className="p-3">{row.line}</td>
                    <td className="p-3">{row.productName}</td>
                    <td className="p-3 font-mono">{row.sku}</td>
                    <td className="p-3">{row.action}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Link
            href={workspacePath(current.slug, "/products")}
            className="mt-3 inline-block text-accent hover:underline"
          >
            View products
          </Link>
        </section>
      )}
    </div>
  );
}
