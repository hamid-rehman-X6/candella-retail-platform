import Link from "next/link";
import { notFound } from "next/navigation";
import { ArrowLeft, ShieldCheck, ShieldOff } from "lucide-react";
import { getPlatformTenant } from "@/lib/platform-api";
import { industryLabel } from "@/lib/industries";
import {
  StatusPill,
  ActivityList,
  maskEmail,
  formatDate,
} from "@/components/platform/ui";

export default async function PlatformTenantDetailPage({
  params,
}: {
  params: Promise<{ tenantId: string }>;
}) {
  const { tenantId } = await params;
  const tenant = await getPlatformTenant(tenantId);
  if (!tenant) notFound();

  const facts: { label: string; value: string }[] = [
    { label: "Slug", value: tenant.slug },
    { label: "Industry", value: industryLabel(tenant.industryType) },
    { label: "Currency", value: tenant.currencyCode },
    { label: "Timezone", value: tenant.timezone },
    { label: "Created", value: formatDate(tenant.createdAt) },
    { label: "Workspace ID", value: tenant.id },
  ];

  return (
    <div className="flex flex-col gap-8">
      <div>
        <Link
          href="/platform/tenants"
          className="inline-flex items-center gap-1.5 text-sm text-slate-400 transition-colors hover:text-slate-200"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          All workspaces
        </Link>

        <div className="mt-3 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight text-white">
            {tenant.name}
          </h1>
          <StatusPill status={tenant.status} />
        </div>
        <p className="mt-1 text-sm text-slate-400">
          {tenant.memberCount} {tenant.memberCount === 1 ? "member" : "members"}
          {tenant.ownerEmail && <> · owner {maskEmail(tenant.ownerEmail)}</>}
        </p>
      </div>

      <dl className="grid gap-px overflow-hidden rounded-xl border border-slate-800 bg-slate-800 sm:grid-cols-2 lg:grid-cols-3">
        {facts.map((f) => (
          <div key={f.label} className="bg-slate-900/60 px-4 py-3">
            <dt className="text-xs tracking-wide text-slate-500 uppercase">
              {f.label}
            </dt>
            <dd className="mt-1 font-mono text-sm break-all text-slate-200">
              {f.value}
            </dd>
          </div>
        ))}
      </dl>

      <section>
        <h2 className="mb-3 text-lg font-semibold tracking-tight text-white">
          Members
        </h2>
        <div className="overflow-x-auto rounded-xl border border-slate-800">
          <table className="w-full min-w-[44rem] text-sm">
            <thead className="bg-slate-900/70 text-left text-xs tracking-wide text-slate-400 uppercase">
              <tr>
                <th className="px-4 py-3 font-medium">Member</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">2FA</th>
                <th className="px-4 py-3 font-medium">Last sign-in</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800">
              {tenant.members.map((m) => (
                <tr key={m.userId} className="bg-slate-900/40">
                  <td className="px-4 py-3">
                    <span className="block font-medium text-slate-100">
                      {m.fullName}
                    </span>
                    <span className="block text-xs text-slate-500">
                      {maskEmail(m.email)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-slate-300 capitalize">
                    {m.role}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={m.status} />
                  </td>
                  <td className="px-4 py-3">
                    {m.mfaEnabled ? (
                      <span className="inline-flex items-center gap-1.5 text-emerald-400">
                        <ShieldCheck className="h-4 w-4" /> On
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 text-slate-500">
                        <ShieldOff className="h-4 w-4" /> Off
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-xs text-slate-500">
                    {m.lastLoginAt ? formatDate(m.lastLoginAt) : "Never"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold tracking-tight text-white">
          Activity
        </h2>
        <ActivityList entries={tenant.recentActivity} />
      </section>
    </div>
  );
}
