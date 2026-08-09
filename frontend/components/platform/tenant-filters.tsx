"use client";

import { useRouter } from "next/navigation";
import { Search } from "lucide-react";
import { industries } from "@/lib/industries";

const statuses = ["active", "trialing", "suspended", "cancelled"];

const field =
  "rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200 outline-none transition-colors focus:border-rose-500/60";

/**
 * Filters submit as a plain GET form, so the result is a real URL: staff can
 * bookmark a filtered view or paste it into a ticket and it resolves identically.
 * Paging links read the same query params.
 */
export function TenantFilters({
  search,
  status,
  industry,
}: {
  search: string;
  status: string;
  industry: string;
}) {
  const router = useRouter();

  return (
    <form
      action="/platform/tenants"
      className="flex flex-wrap items-center gap-2"
    >
      <div className="relative min-w-56 flex-1">
        <Search className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-500" />
        <input
          type="search"
          name="search"
          defaultValue={search}
          placeholder="Search by name or slug"
          className={`${field} w-full pl-9 placeholder:text-slate-500`}
        />
      </div>

      <select name="status" defaultValue={status} className={field}>
        <option value="">Any status</option>
        {statuses.map((s) => (
          <option key={s} value={s}>
            {s.charAt(0).toUpperCase() + s.slice(1)}
          </option>
        ))}
      </select>

      <select name="industry" defaultValue={industry} className={field}>
        <option value="">Any industry</option>
        {industries.map((i) => (
          <option key={i.value} value={i.value}>
            {i.label}
          </option>
        ))}
      </select>

      <button
        type="submit"
        className="rounded-lg bg-rose-500/90 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-rose-500"
      >
        Apply
      </button>

      {(search || status || industry) && (
        <button
          type="button"
          onClick={() => router.push("/platform/tenants")}
          className="rounded-lg border border-slate-700 px-3 py-2 text-sm text-slate-400 transition-colors hover:text-slate-200"
        >
          Clear
        </button>
      )}
    </form>
  );
}
