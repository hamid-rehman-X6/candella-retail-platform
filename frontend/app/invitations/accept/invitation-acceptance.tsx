"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import {
  getMe,
  getWorkspaces,
  pendingInvitation,
  type AuthUser,
} from "@/lib/auth-api";
import { acceptStaffInvitation } from "@/lib/staff-api";

export function InvitationAcceptance({ token }: { token: string }) {
  const router = useRouter();
  const [user, setUser] = useState<AuthUser | null>(null);
  const [loading, setLoading] = useState(Boolean(token));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!token) {
      return;
    }
    pendingInvitation.set(token);
    let cancelled = false;
    void getMe()
      .then(({ user }) => {
        if (!cancelled) setUser(user);
      })
      .catch(() => {
        if (!cancelled) setUser(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function accept() {
    if (!token) return;
    setBusy(true);
    setError("");
    try {
      const result = await acceptStaffInvitation(token);
      pendingInvitation.clear();
      const { workspaces } = await getWorkspaces();
      const workspace = workspaces.find((item) => item.id === result.tenantId);
      router.replace(workspace ? `/w/${workspace.slug}` : "/dashboard");
      router.refresh();
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "Could not accept this invitation.",
      );
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-canvas px-4 py-10">
      <section className="w-full max-w-lg rounded-2xl border border-line bg-surface p-8">
        <p className="text-xs font-semibold tracking-widest text-accent uppercase">
          Candella
        </p>
        <h1 className="mt-3 text-2xl font-semibold">Join a workspace</h1>
        {!token ? (
          <p role="alert" className="mt-4 text-sm text-red-300">
            This invitation link is missing its token. Ask the owner to resend
            it.
          </p>
        ) : loading ? (
          <p className="mt-4 text-sm text-muted">Checking your account...</p>
        ) : !user ? (
          <div className="mt-4 space-y-4 text-sm">
            <p className="text-muted">
              Sign in or create an account using the email address that received
              this invitation. You will return here afterward.
            </p>
            <div className="flex gap-3">
              <Link
                className="rounded-xl bg-accent px-4 py-2.5 font-semibold text-black"
                href="/login"
              >
                Sign in
              </Link>
              <Link
                className="rounded-xl border border-line px-4 py-2.5"
                href="/signup"
              >
                Create account
              </Link>
            </div>
          </div>
        ) : (
          <div className="mt-4 space-y-4 text-sm">
            <p className="text-muted">
              Signed in as{" "}
              <span className="font-medium text-foreground">{user.email}</span>.
              The invitation can only be accepted by this verified email
              address.
            </p>
            {!user.emailVerified && (
              <p role="alert" className="text-red-300">
                Verify your email before accepting.
              </p>
            )}
            {error && (
              <p role="alert" className="text-red-300">
                {error}
              </p>
            )}
            <button
              type="button"
              disabled={busy || !user.emailVerified}
              onClick={() => void accept()}
              className="rounded-xl bg-accent px-4 py-2.5 font-semibold text-black disabled:opacity-50"
            >
              {busy ? "Joining..." : "Accept invitation"}
            </button>
          </div>
        )}
      </section>
    </main>
  );
}
