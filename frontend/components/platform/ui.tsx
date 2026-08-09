import type { ActivityEntry } from "@/lib/platform-api";
import { cn } from "@/lib/cn";

/** A headline number. */
export function StatCard({
  label,
  value,
  hint,
}: {
  label: string;
  value: string | number;
  hint?: string;
}) {
  return (
    <div className="rounded-xl border border-slate-800 bg-slate-900/50 p-5">
      <p className="text-sm text-slate-400">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-tight text-white">
        {value}
      </p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  );
}

const statusTone: Record<string, string> = {
  active: "bg-emerald-500/15 text-emerald-300 ring-emerald-500/25",
  trialing: "bg-sky-500/15 text-sky-300 ring-sky-500/25",
  suspended: "bg-amber-500/15 text-amber-300 ring-amber-500/25",
  cancelled: "bg-slate-500/15 text-slate-300 ring-slate-500/25",
  invited: "bg-sky-500/15 text-sky-300 ring-sky-500/25",
  removed: "bg-slate-500/15 text-slate-300 ring-slate-500/25",
};

export function StatusPill({ status }: { status: string }) {
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium capitalize ring-1",
        statusTone[status] ??
          "bg-slate-500/15 text-slate-300 ring-slate-500/25",
      )}
    >
      {status}
    </span>
  );
}

/**
 * Masks an email down to its first character and domain. Support staff rarely
 * need the raw address to do their job, and defaulting to masked means a shared
 * screen or a screenshot doesn't leak a customer's contact details.
 */
export function maskEmail(email: string): string {
  const [local = "", domain = ""] = email.split("@");
  if (!domain) return email;
  return `${local.charAt(0)}${"•".repeat(Math.max(local.length - 1, 1))}@${domain}`;
}

/** Absolute UTC timestamp — unambiguous for staff reading across time zones. */
export function formatDate(iso: string): string {
  return new Date(iso).toISOString().replace("T", " ").slice(0, 16) + " UTC";
}

export function ActivityList({ entries }: { entries: ActivityEntry[] }) {
  if (entries.length === 0) {
    return (
      <p className="rounded-xl border border-dashed border-slate-800 px-4 py-8 text-center text-sm text-slate-500">
        No activity recorded yet.
      </p>
    );
  }

  return (
    <ul className="divide-y divide-slate-800 overflow-hidden rounded-xl border border-slate-800">
      {entries.map((e) => (
        <li
          key={e.id}
          className="flex flex-wrap items-baseline gap-x-3 gap-y-1 bg-slate-900/40 px-4 py-3 text-sm"
        >
          <code className="font-mono text-xs text-rose-300">{e.action}</code>
          {e.actorType === "platform_admin" && (
            <span className="rounded bg-rose-500/15 px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-rose-300 uppercase">
              Staff
            </span>
          )}
          <span className="text-slate-400">
            {e.actorEmail ? maskEmail(e.actorEmail) : "system"}
          </span>
          {e.tenantName && (
            <span className="text-slate-500">· {e.tenantName}</span>
          )}
          <span className="ml-auto text-xs text-slate-500">
            {formatDate(e.createdAt)}
          </span>
        </li>
      ))}
    </ul>
  );
}
