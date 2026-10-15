"use client";

import { Select } from "@/components/ui/select";

import { useCallback, useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { useApp } from "@/components/app/app-context";
import { formatDate } from "@/lib/datetime";
import {
  changeStaffRole,
  inviteStaff,
  listStaff,
  removeStaff,
  resendStaffInvite,
  revokeStaffInvite,
  type StaffInvitation,
  type StaffMember,
} from "@/lib/staff-api";

const roles = ["member", "admin", "owner"] as const;

export default function StaffPage() {
  const { current, user } = useApp();
  const router = useRouter();
  const [members, setMembers] = useState<StaffMember[]>([]);
  const [invitations, setInvitations] = useState<StaffInvitation[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<StaffMember["role"]>("member");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(async () => {
    const result = await listStaff(current.id);
    setMembers(result.members);
    setInvitations(result.invitations);
  }, [current.id]);

  useEffect(() => {
    let cancelled = false;
    void listStaff(current.id)
      .then((result) => {
        if (!cancelled) {
          setMembers(result.members);
          setInvitations(result.invitations);
        }
      })
      .catch((cause) => {
        if (!cancelled)
          setError(
            cause instanceof Error ? cause.message : "Could not load staff.",
          );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [current.id]);

  async function action(work: () => Promise<unknown>, success: string) {
    setBusy(true);
    setError("");
    setNotice("");
    try {
      await work();
      await load();
      setNotice(success);
      router.refresh();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "The change could not be saved.",
      );
      // An invitation can be saved even when email delivery fails.
      void load().catch(() => undefined);
    } finally {
      setBusy(false);
    }
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    void action(async () => {
      await inviteStaff(current.id, email.trim(), role);
      setEmail("");
    }, "Invitation sent. The recipient can sign in or create an account with that email.");
  }

  if (current.role !== "owner" && current.role !== "admin") {
    return (
      <p role="alert" className="text-sm text-muted">
        Only workspace owners and admins can manage staff.
      </p>
    );
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6 pb-10">
      <div>
        <h1 className="text-3xl font-semibold tracking-tight">Staff</h1>
        <p className="mt-1 text-sm text-muted">
          Invite cashiers and managers, change roles, or remove access to{" "}
          {current.name}.
        </p>
      </div>

      {error && (
        <p
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 p-3 text-sm text-danger-ink"
        >
          {error}
        </p>
      )}
      {notice && (
        <p
          role="status"
          className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-success-ink"
        >
          {notice}
        </p>
      )}

      <form
        onSubmit={submit}
        className="grid gap-3 rounded-2xl border border-line bg-surface p-5 sm:grid-cols-[minmax(0,1fr)_160px_auto] sm:items-end"
      >
        <label className="text-sm font-medium">
          Invite by email
          <input
            required
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="cashier@example.com"
            className="mt-2 w-full rounded-xl border border-line bg-canvas px-3 py-2.5 outline-none focus:border-accent"
          />
        </label>
        <label className="text-sm font-medium">
          Role
          <Select
            ariaLabel="Role"
            value={role}
            onValueChange={(value) => setRole(value as StaffMember["role"])}
            className="mt-2 w-full rounded-xl border border-line bg-canvas px-3 py-2.5 outline-none focus:border-accent"
            options={roles
              .filter((item) => current.role === "owner" || item !== "owner")
              .map((item) => ({ value: item, label: item }))}
          />
        </label>
        <button
          disabled={busy}
          className="rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black disabled:opacity-50"
        >
          Send invite
        </button>
      </form>

      <section className="rounded-2xl border border-line bg-surface p-5">
        <h2 className="text-lg font-semibold">Members</h2>
        {loading ? (
          <p className="mt-4 text-sm text-muted">Loading staff...</p>
        ) : members.length === 0 ? (
          <p className="mt-4 text-sm text-muted">No staff found.</p>
        ) : (
          <div className="mt-4 divide-y divide-line">
            {members.map((member) => {
              const protectedOwner =
                current.role !== "owner" && member.role === "owner";
              const protectedSelf =
                current.role === "admin" && member.userId === user.id;
              const canEdit = !protectedOwner && !protectedSelf;
              return (
                <div
                  key={member.userId}
                  className="flex flex-wrap items-center gap-3 py-4"
                >
                  <div className="min-w-0 flex-1">
                    <p className="font-medium">
                      {member.fullName}
                      {member.userId === user.id ? " (you)" : ""}
                    </p>
                    <p className="truncate text-sm text-muted">
                      {member.email} · {member.status}
                    </p>
                  </div>
                  <Select
                    ariaLabel={`Role for ${member.fullName}`}
                    value={member.role}
                    disabled={busy || !canEdit || member.status !== "active"}
                    onValueChange={(value) =>
                      void action(
                        () =>
                          changeStaffRole(
                            current.id,
                            member.userId,
                            value as StaffMember["role"],
                          ),
                        "Role updated.",
                      )
                    }
                    className="rounded-lg border border-line bg-canvas px-3 py-2 text-sm disabled:opacity-50"
                    options={roles
                      .filter(
                        (item) => current.role === "owner" || item !== "owner",
                      )
                      .map((item) => ({ value: item, label: item }))}
                  />
                  <button
                    type="button"
                    disabled={busy || !canEdit || member.status !== "active"}
                    onClick={() => {
                      if (
                        window.confirm(
                          `Remove ${member.fullName} from this workspace? Their access stops immediately.`,
                        )
                      )
                        void action(
                          () => removeStaff(current.id, member.userId),
                          "Staff access removed.",
                        );
                    }}
                    className="rounded-lg border border-line px-3 py-2 text-sm text-danger-ink disabled:opacity-40"
                  >
                    Remove
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </section>

      <section className="rounded-2xl border border-line bg-surface p-5">
        <h2 className="text-lg font-semibold">Pending invitations</h2>
        {invitations.length === 0 ? (
          <p className="mt-4 text-sm text-muted">No pending invitations.</p>
        ) : (
          <div className="mt-4 divide-y divide-line">
            {invitations.map((invitation) => (
              <div
                key={invitation.id}
                className="flex flex-wrap items-center gap-3 py-4"
              >
                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{invitation.email}</p>
                  <p className="text-sm text-muted">
                    {invitation.role} · Expires{" "}
                    {formatDate(invitation.expiresAt, current.timezone)}
                  </p>
                </div>
                <button
                  type="button"
                  disabled={
                    busy ||
                    (current.role !== "owner" && invitation.role === "owner")
                  }
                  onClick={() =>
                    void action(
                      () => resendStaffInvite(current.id, invitation.id),
                      "A fresh invitation link was sent.",
                    )
                  }
                  className="rounded-lg border border-line px-3 py-2 text-sm disabled:opacity-40"
                >
                  Resend
                </button>
                <button
                  type="button"
                  disabled={
                    busy ||
                    (current.role !== "owner" && invitation.role === "owner")
                  }
                  onClick={() =>
                    void action(
                      () => revokeStaffInvite(current.id, invitation.id),
                      "Invitation revoked.",
                    )
                  }
                  className="rounded-lg border border-line px-3 py-2 text-sm text-danger-ink disabled:opacity-40"
                >
                  Revoke
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
