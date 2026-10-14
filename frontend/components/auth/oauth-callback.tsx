"use client";

import { useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { LogoMark } from "@/components/ui/logo";
import {
  getMe,
  mfaToken as mfaTokenStore,
  pendingInvitation,
} from "@/lib/auth-api";
import { setSessionUser } from "@/lib/session-store";
import { authErrorCopy } from "@/lib/auth-errors";
import { firstName } from "@/lib/user-name";
import { toast } from "@/lib/toast";

type Props = {
  status: string;
  event?: string;
  reason?: string;
  mfaChallenge?: string;
};

/**
 * The landing point for every Google sign-in outcome.
 *
 * The backend redirects here instead of straight to /dashboard, which gives the
 * flow a visible, branded moment and one place to decide what to say and where
 * to go. The query params are messaging only — a successful outcome is
 * re-confirmed against /auth/me, so a crafted link can't fake a signed-in state.
 */
export function OAuthCallback({ status, event, reason, mfaChallenge }: Props) {
  const router = useRouter();
  const handled = useRef(false);

  useEffect(() => {
    if (handled.current) return; // Strict Mode invokes effects twice in dev.
    handled.current = true;

    // Google covered the first factor only — the account still owes us a code.
    // Move the challenge out of the URL and into sessionStorage immediately.
    if (status === "mfa" && mfaChallenge) {
      mfaTokenStore.set(mfaChallenge);
      router.replace("/two-factor");
      return;
    }

    if (status === "cancelled") {
      toast.info("Sign-in cancelled", {
        description: "No problem — you can try again whenever you're ready.",
      });
      router.replace("/login");
      return;
    }

    if (status !== "ok") {
      const copy = authErrorCopy(reason);
      toast.error(copy.title, { description: copy.description });
      router.replace(
        `/login?error=${encodeURIComponent(reason ?? "google_failed")}`,
      );
      return;
    }

    getMe()
      .then(({ user }) => {
        setSessionUser(user); // so the next page renders with the right identity
        const description = `Signed in with Google as ${user.email}`;

        if (event === "signup") {
          toast.success(`Welcome to Candella, ${firstName(user.fullName)}`, {
            description,
          });
          router.replace(
            pendingInvitation.destination("/onboarding?welcome=google"),
          );
        } else {
          toast.success(`Welcome back, ${firstName(user.fullName)}`, {
            description,
          });
          router.replace(pendingInvitation.destination("/dashboard"));
        }
      })
      .catch(() => {
        const copy = authErrorCopy("google_failed");
        toast.error(copy.title, {
          description:
            "Your session didn't stick. Check that cookies are enabled, then try again.",
        });
        router.replace("/login?error=google_failed");
      });
  }, [status, event, reason, mfaChallenge, router]);

  const message =
    status === "cancelled"
      ? "Taking you back…"
      : status === "ok" || status === "mfa"
        ? "Signing you in…"
        : "One moment…";

  return (
    <div className="flex flex-col items-center text-center">
      <LogoMark className="h-12 w-12 rounded-2xl" />
      <div className="mt-8 flex items-center gap-2.5 text-muted">
        <Loader2 className="h-4 w-4 animate-spin text-accent-ink" />
        <p className="text-sm" aria-live="polite">
          {message}
        </p>
      </div>
    </div>
  );
}
