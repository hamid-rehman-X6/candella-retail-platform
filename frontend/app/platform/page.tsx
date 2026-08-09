import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { getPlatformOverview } from "@/lib/platform-api";
import { StatCard, ActivityList } from "@/components/platform/ui";

export default async function PlatformOverviewPage() {
  const overview = await getPlatformOverview();

  if (!overview) {
    return (
      <p className="rounded-xl border border-slate-800 bg-slate-900/50 p-6 text-sm text-slate-400">
        Couldn&apos;t load platform metrics. Check that the API is reachable.
      </p>
    );
  }

  const { tenants } = overview;

  return (
    <div className="flex flex-col gap-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight text-white">
          Platform overview
        </h1>
        <p className="mt-1 text-sm text-slate-400">
          Every workspace and account on Candella.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Workspaces"
          value={tenants.total}
          hint={`+${overview.newTenants7d} in the last 7 days`}
        />
        <StatCard
          label="Users"
          value={overview.totalUsers}
          hint={`+${overview.newUsers7d} in the last 7 days`}
        />
        <StatCard
          label="Active sessions"
          value={overview.activeSessions}
          hint="Signed in right now"
        />
        <StatCard
          label="Needs attention"
          value={tenants.suspended}
          hint="Suspended workspaces"
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-4">
        <StatCard label="Active" value={tenants.active} />
        <StatCard label="Trialing" value={tenants.trialing} />
        <StatCard label="Suspended" value={tenants.suspended} />
        <StatCard label="Cancelled" value={tenants.cancelled} />
      </div>

      <div>
        <div className="mb-3 flex items-baseline justify-between">
          <h2 className="text-lg font-semibold tracking-tight text-white">
            Recent activity
          </h2>
          <Link
            href="/platform/tenants"
            className="inline-flex items-center gap-1 text-sm text-rose-300 transition-colors hover:text-rose-200"
          >
            Browse workspaces
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>
        <ActivityList entries={overview.recentActivity} />
      </div>
    </div>
  );
}
