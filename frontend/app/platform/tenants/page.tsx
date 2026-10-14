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
        <h1 className="text-2xl font-semibold tracking-tight text-foreground">
          Workspaces
        </h1>
        <p className="mt-1 text-sm text-muted">
          {total} {total === 1 ? "workspace" : "workspaces"} on the platform.
        </p>
      </div>

      <TenantFilters
        key={JSON.stringify([search, status, industry])}
        search={search}
        status={status}
        industry={industry}
      />

      {tenants.length === 0 ? (
        <p className="rounded-xl border border-dashed border-line px-4 py-12 text-center text-sm text-subtle">
          No workspaces match these filters.
        </p>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-line">
          <table className="w-full min-w-[52rem] text-sm">
            <thead className="bg-surface-2/70 text-left text-xs tracking-wide text-muted uppercase">
              <tr>
                <th className="px-4 py-3 font-medium">Workspace</th>
                <th className="px-4 py-3 font-medium">Industry</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Owner</th>
                <th className="px-4 py-3 text-right font-medium">Members</th>
                <th className="px-4 py-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {tenants.map((t) => (
                <tr
                  key={t.id}
                  className="bg-surface/40 transition-colors hover:bg-surface-2"
                >
                  <td className="px-4 py-3">
                    <Link
                      href={`/platform/tenants/${t.id}`}
                      className="font-medium text-foreground hover:text-platform-accent"
                    >
                      {t.name}
                    </Link>
                    <span className="block font-mono text-xs text-subtle">
                      {t.slug}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted">
                    {industryLabel(t.industryType)}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={t.status} />
                  </td>
                  <td className="px-4 py-3 text-muted">
                    {t.ownerEmail ? maskEmail(t.ownerEmail) : "—"}
                  </td>
                  <td className="px-4 py-3 text-right text-muted">
                    {t.memberCount}
                  </td>
                  <td className="px-4 py-3 text-xs text-subtle">
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
          <span className="text-subtle">
            Page {page} of {lastPage}
          </span>
          <div className="flex gap-2">
            {page > 1 && (
              <Link
                href={pageHref(page - 1)}
                className="rounded-lg border border-line-strong px-3 py-1.5 text-muted transition-colors hover:border-line-strong hover:text-foreground"
              >
                Previous
              </Link>
            )}
            {page < lastPage && (
              <Link
                href={pageHref(page + 1)}
                className="rounded-lg border border-line-strong px-3 py-1.5 text-muted transition-colors hover:border-line-strong hover:text-foreground"
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
