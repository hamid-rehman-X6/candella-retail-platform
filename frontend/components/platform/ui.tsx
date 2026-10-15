import type { ActivityEntry } from "@/lib/platform-api";
import { cn } from "@/lib/cn";
import { DEFAULT_TIMEZONE, formatDateTime } from "@/lib/datetime";

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
    <div className="rounded-xl border border-line bg-surface/50 p-5">
      <p className="text-sm text-muted">{label}</p>
      <p className="mt-2 text-3xl font-semibold tracking-tight text-foreground">
        {value}
      </p>
      {hint && <p className="mt-1 text-xs text-subtle">{hint}</p>}
    </div>
  );
}

const statusTone: Record<string, string> = {
  active: "bg-emerald-500/15 text-success-ink ring-emerald-500/25",
  trialing: "bg-sky-500/15 text-info-ink ring-sky-500/25",
  suspended: "bg-amber-500/15 text-warning-ink ring-amber-500/25",
  cancelled: "bg-slate-500/15 text-muted ring-slate-500/25",
  invited: "bg-sky-500/15 text-info-ink ring-sky-500/25",
  removed: "bg-slate-500/15 text-muted ring-slate-500/25",
};

export function StatusPill({ status }: { status: string }) {
  return (
    <span
      className={cn(
        "inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium capitalize ring-1",
        statusTone[status] ?? "bg-slate-500/15 text-muted ring-slate-500/25",
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

/**
 * Pakistan time with an explicit zone label. Staff look at many workspaces at
 * once, so the label keeps a timestamp unambiguous even when a workspace has
 * been set to another zone.
 */
export function formatDate(iso: string): string {
  return `${formatDateTime(iso, DEFAULT_TIMEZONE)} PKT`;
}

export function ActivityList({ entries }: { entries: ActivityEntry[] }) {
  if (entries.length === 0) {
    return (
      <p className="rounded-xl border border-dashed border-line px-4 py-8 text-center text-sm text-subtle">
        No activity recorded yet.
      </p>
    );
  }

  return (
    <ul className="divide-y divide-line overflow-hidden rounded-xl border border-line">
      {entries.map((e) => (
        <li
          key={e.id}
          className="flex flex-wrap items-baseline gap-x-3 gap-y-1 bg-surface/40 px-4 py-3 text-sm"
        >
          <code className="font-mono text-xs text-platform-accent">
            {e.action}
          </code>
          {e.actorType === "platform_admin" && (
            <span className="rounded bg-rose-500/15 px-1.5 py-0.5 text-[10px] font-medium tracking-wide text-platform-accent uppercase">
              Staff
            </span>
          )}
          <span className="text-muted">
            {e.actorEmail ? maskEmail(e.actorEmail) : "system"}
          </span>
          {e.tenantName && (
            <span className="text-subtle">· {e.tenantName}</span>
          )}
          <span className="ml-auto text-xs text-subtle">
            {formatDate(e.createdAt)}
          </span>
        </li>
      ))}
    </ul>
  );
}
