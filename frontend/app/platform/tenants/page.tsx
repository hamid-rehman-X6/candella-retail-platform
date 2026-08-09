import Link from "next/link";
import { getPlatformTenants } from "@/lib/platform-api";
import { industryLabel } from "@/lib/industries";
import { StatusPill, maskEmail, formatDate } from "@/components/platform/ui";
import { TenantFilters } from "@/components/platform/tenant-filters";

const PAGE_SIZE = 25;

export default async function PlatformTenantsPage({
  searchParams,
}: {
  searchParams: Promise<{ [key: string]: string | string[] | undefined }>;
}) {
  const params = await searchParams;
  const first = (key: string) => {
    const raw = params[key];
    return Array.isArray(raw) ? raw[0] : raw;
  };

  const page = Math.max(1, Number(first("page") ?? 1) || 1);
  const search = first("search") ?? "";
  const status = first("status") ?? "";
  const industry = first("industry") ?? "";

  const { tenants, total } = await getPlatformTenants({
    search,
    status,
    industry,
    limit: PAGE_SIZE,
    offset: (page - 1) * PAGE_SIZE,
  });

  const lastPage = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const pageHref = (n: number) => {
    const q = new URLSearchParams();
    if (search) q.set("search", search);
    if (status) q.set("status", status);
    if (industry) q.set("industry", industry);
    if (n > 1) q.set("page", String(n));
    const qs = q.toString();
    return qs ? `/platform/tenants?${qs}` : "/platform/tenants";
  };

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-white">
          Workspaces
        </h1>
        <p className="mt-1 text-sm text-slate-400">
          {total} {total === 1 ? "workspace" : "workspaces"} on the platform.
        </p>
      </div>

      <TenantFilters search={search} status={status} industry={industry} />

      {tenants.length === 0 ? (
        <p className="rounded-xl border border-dashed border-slate-800 px-4 py-12 text-center text-sm text-slate-500">
          No workspaces match these filters.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-slate-800">
          <table className="w-full min-w-[52rem] text-sm">
            <thead className="bg-slate-900/70 text-left text-xs tracking-wide text-slate-400 uppercase">
              <tr>
                <th className="px-4 py-3 font-medium">Workspace</th>
                <th className="px-4 py-3 font-medium">Industry</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Owner</th>
                <th className="px-4 py-3 text-right font-medium">Members</th>
                <th className="px-4 py-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {tenants.map((t) => (
                <tr
                  key={t.id}
                  className="bg-slate-900/40 transition-colors hover:bg-slate-900"
                >
                  <td className="px-4 py-3">
                    <Link
                      href={`/platform/tenants/${t.id}`}
                      className="font-medium text-white hover:text-rose-300"
                    >
                      {t.name}
                    </Link>
                    <span className="block font-mono text-xs text-slate-500">
                      {t.slug}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-300">
                    {industryLabel(t.industryType)}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={t.status} />
                  </td>
                  <td className="px-4 py-3 text-slate-400">
                    {t.ownerEmail ? maskEmail(t.ownerEmail) : "—"}
                  </td>
                  <td className="px-4 py-3 text-right text-slate-300">
                    {t.memberCount}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-500">
                    {formatDate(t.createdAt)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {lastPage > 1 && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-slate-500">
            Page {page} of {lastPage}
          </span>
          <div className="flex gap-2">
            {page > 1 && (
              <Link
                href={pageHref(page - 1)}
                className="rounded-lg border border-slate-700 px-3 py-1.5 text-slate-300 transition-colors hover:border-slate-600 hover:text-white"
              >
                Previous
              </Link>
            )}
            {page < lastPage && (
              <Link
                href={pageHref(page + 1)}
                className="rounded-lg border border-slate-700 px-3 py-1.5 text-slate-300 transition-colors hover:border-slate-600 hover:text-white"
              >
                Next
              </Link>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
