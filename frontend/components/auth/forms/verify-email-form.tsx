"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { MailCheck, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { OtpInput } from "@/components/auth/otp-input";
import { Button } from "@/components/ui/button";
import { ApiError } from "@/lib/api";
import { verifyEmail, resendVerification, pendingEmail } from "@/lib/auth-api";
import { useSessionValue } from "@/lib/use-session-value";

const RESEND_SECONDS = 30;

export function VerifyEmailForm() {
  const router = useRouter();
  // The email was stashed by the signup / login step (client-only, hydration-safe).
  const email = useSessionValue(pendingEmail.get);
  const [code, setCode] = useState("");
  const [error, setError] = useState<string>();
  const [verifying, setVerifying] = useState(false);
  const [seconds, setSeconds] = useState(RESEND_SECONDS);
  const [resent, setResent] = useState(false);

  useEffect(() => {
    if (seconds <= 0) return;
    const id = setTimeout(() => setSeconds((s) => s - 1), 1000);
    return () => clearTimeout(id);
  }, [seconds]);

  async function verify(value: string) {
    setVerifying(true);
    setError(undefined);
    try {
      await verifyEmail({ email, code: value });
      pendingEmail.clear();
      // Verified + signed in — prompt to secure the account with 2FA.
      router.push("/two-factor/setup");
    } catch (err) {
      setVerifying(false);
      setCode("");
      setError(
        err instanceof ApiError
          ? err.message
          : "Something went wrong. Try again.",
      );
    }
  }

  async function resend() {
    setSeconds(RESEND_SECONDS);
    setCode("");
    setError(undefined);
    try {
      await resendVerification(email);
      setResent(true);
    } catch {
      /* generic — no enumeration; nothing to show */
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        icon={MailCheck}
        title="Verify your email"
        subtitle={
          email ? (
            <>
              Enter the 6-digit code we sent to{" "}
              <span className="font-medium text-foreground/90">{email}</span>.
            </>
          ) : (
            "Enter the 6-digit code we sent to your email address."
          )
        }
      />

      <div className="flex flex-col gap-4">
        <OtpInput
          value={code}
          onChange={setCode}
          onComplete={verify}
          error={!!error}
          autoFocus
          disabled={verifying}
        />
        {error && <p className="text-sm text-red-400">{error}</p>}
        {resent && !error && (
          <p className="text-sm text-emerald-400">
            A fresh code is on its way.
          </p>
        )}

        <Button
          size="lg"
          className="mt-2 w-full"
          disabled={code.length < 6 || verifying}
          onClick={() => verify(code)}
        >
          {verifying ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Verifying…
            </>
          ) : (
            <>
              Verify email
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>

        <p className="text-center text-sm text-muted">
          Didn&apos;t get it?{" "}
          {seconds > 0 ? (
            <span className="text-subtle">Resend in {seconds}s</span>
          ) : (
            <button
              type="button"
              onClick={resend}
              className="font-medium text-accent transition-colors hover:text-accent-bright"
            >
              Resend code
            </button>
          )}
        </p>
      </div>

      <p className="text-center text-sm text-muted">
        Wrong address?{" "}
        <Link
          href="/signup"
          className="font-medium text-accent transition-colors hover:text-accent-bright"
        >
          Change email
        </Link>
      </p>
    </div>
  );
}
