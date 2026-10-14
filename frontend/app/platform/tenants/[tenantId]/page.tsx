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
          className="inline-flex items-center gap-1.5 text-sm text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          All workspaces
        </Link>

        <div className="mt-3 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight text-foreground">
            {tenant.name}
          </h1>
          <StatusPill status={tenant.status} />
        </div>
        <p className="mt-1 text-sm text-muted">
          {tenant.memberCount} {tenant.memberCount === 1 ? "member" : "members"}
          {tenant.ownerEmail && <> · owner {maskEmail(tenant.ownerEmail)}</>}
        </p>
      </div>

      <dl className="grid gap-px overflow-hidden rounded-xl border border-line bg-surface-2 sm:grid-cols-2 lg:grid-cols-3">
        {facts.map((f) => (
          <div key={f.label} className="bg-surface/60 px-4 py-3">
            <dt className="text-xs tracking-wide text-subtle uppercase">
              {f.label}
            </dt>
            <dd className="mt-1 font-mono text-sm break-all text-foreground">
              {f.value}
            </dd>
          </div>
        ))}
      </dl>

      <section>
        <h2 className="mb-3 text-lg font-semibold tracking-tight text-foreground">
          Members
        </h2>
        <div className="overflow-x-auto rounded-xl border border-line">
          <table className="w-full min-w-[44rem] text-sm">
            <thead className="bg-surface-2/70 text-left text-xs tracking-wide text-muted uppercase">
              <tr>
                <th className="px-4 py-3 font-medium">Member</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">2FA</th>
                <th className="px-4 py-3 font-medium">Last sign-in</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-line">
              {tenant.members.map((m) => (
                <tr key={m.userId} className="bg-surface/40">
                  <td className="px-4 py-3">
                    <span className="block font-medium text-foreground">
                      {m.fullName}
                    </span>
                    <span className="block text-xs text-subtle">
                      {maskEmail(m.email)}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-muted capitalize">{m.role}</td>
                  <td className="px-4 py-3">
                    <StatusPill status={m.status} />
                  </td>
                  <td className="px-4 py-3">
                    {m.mfaEnabled ? (
                      <span className="inline-flex items-center gap-1.5 text-success-ink">
                        <ShieldCheck className="h-4 w-4" /> On
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 text-subtle">
                        <ShieldOff className="h-4 w-4" /> Off
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-xs text-subtle">
                    {m.lastLoginAt ? formatDate(m.lastLoginAt) : "Never"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section>
        <h2 className="mb-3 text-lg font-semibold tracking-tight text-foreground">
          Activity
        </h2>
        <ActivityList entries={tenant.recentActivity} />
      </section>
    </div>
  );
}
