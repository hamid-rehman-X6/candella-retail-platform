"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import {
  ShieldCheck,
  Copy,
  Check,
  ArrowRight,
  Loader2,
  CircleCheck,
} from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { OtpInput } from "@/components/auth/otp-input";
import { Button } from "@/components/ui/button";
import { CheckLine } from "@/components/auth/brand-panel";
import { ApiError } from "@/lib/api";
import { setupTwoFactor, enableTwoFactor } from "@/lib/auth-api";

type Step = "loading" | "scan" | "confirm" | "done";

export function TwoFactorSetupForm() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("loading");
  const [secret, setSecret] = useState("");
  const [otpauthUrl, setOtpauthUrl] = useState("");
  const [backupCodes, setBackupCodes] = useState<string[]>([]);
  const [code, setCode] = useState("");
  const [error, setError] = useState<string>();
  const [verifying, setVerifying] = useState(false);
  const [copied, setCopied] = useState(false);

  // Fetch a fresh secret + QR when the page opens (requires a session).
  useEffect(() => {
    let active = true;
    setupTwoFactor()
      .then((res) => {
        if (!active) return;
        setSecret(res.secret);
        setOtpauthUrl(res.otpauthUrl);
        setStep("scan");
      })
      .catch((err) => {
        if (!active) return;
        if (err instanceof ApiError && err.code === "unauthenticated") {
          router.replace("/login");
          return;
        }
        if (err instanceof ApiError && err.code === "mfa_already_enabled") {
          router.replace("/dashboard");
          return;
        }
        setError(
          err instanceof ApiError ? err.message : "Couldn't start 2FA setup.",
        );
        setStep("scan");
      });
    return () => {
      active = false;
    };
  }, [router]);

  async function copySecret() {
    try {
      await navigator.clipboard.writeText(secret);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  async function confirm(value: string) {
    setVerifying(true);
    setError(undefined);
    try {
      const res = await enableTwoFactor(value);
      setBackupCodes(res.backupCodes);
      setStep("done");
    } catch (err) {
      setVerifying(false);
      setCode("");
      setError(err instanceof ApiError ? err.message : "Something went wrong.");
    }
  }

  if (step === "loading") {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-muted">
        <Loader2 className="h-6 w-6 animate-spin text-accent" />
        Preparing two-factor setup…
      </div>
    );
  }

  if (step === "done") {
    return (
      <div className="flex flex-col gap-8">
        <AuthHeader
          icon={CircleCheck}
          title="Two-factor is on"
          subtitle="Save these backup codes somewhere safe. Each one works once if you ever lose access to your authenticator app."
        />

        <div className="grid grid-cols-2 gap-2 rounded-2xl border border-line bg-white/[0.02] p-4 font-mono text-sm">
          {backupCodes.map((c) => (
            <span key={c} className="text-center text-foreground/85">
              {c}
            </span>
          ))}
        </div>

        <ul className="flex flex-col gap-2">
          <CheckLine>
            Store them in a password manager, not your inbox.
          </CheckLine>
          <CheckLine>Each code can only be used one time.</CheckLine>
        </ul>

        <Button
          size="lg"
          className="w-full"
          onClick={() => router.push("/dashboard")}
        >
          Finish setup
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
      </div>
    );
  }

  if (step === "confirm") {
    return (
      <div className="flex flex-col gap-8">
        <AuthHeader
          icon={ShieldCheck}
          title="Confirm the code"
          subtitle="Enter the 6-digit code your authenticator app is showing to finish turning on 2FA."
        />

        <div className="flex flex-col gap-4">
          <OtpInput
            value={code}
            onChange={setCode}
            onComplete={confirm}
            error={!!error}
            autoFocus
            disabled={verifying}
          />
          {error && <p className="text-sm text-red-400">{error}</p>}

          <Button
            size="lg"
            className="mt-2 w-full"
            disabled={code.length < 6 || verifying}
            onClick={() => confirm(code)}
          >
            {verifying ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Confirming…
              </>
            ) : (
              <>
                Enable 2FA
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </>
            )}
          </Button>

          <button
            type="button"
            onClick={() => {
              setStep("scan");
              setError(undefined);
            }}
            className="text-center text-sm text-muted transition-colors hover:text-accent"
          >
            Back to QR code
          </button>
        </div>
      </div>
    );
  }

  // step === "scan"
  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        icon={ShieldCheck}
        title="Set up two-factor auth"
        subtitle="Scan this QR code with an authenticator app like Google Authenticator, 1Password or Authy."
      />

      {error && <p className="text-sm text-red-400">{error}</p>}

      <div className="flex flex-col items-center gap-5">
        <div className="rounded-2xl bg-white p-4 shadow-lg">
          {otpauthUrl ? (
            <QRCodeSVG value={otpauthUrl} size={168} level="M" />
          ) : (
            <div className="flex h-[168px] w-[168px] items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-black/40" />
            </div>
          )}
        </div>

        <div className="w-full">
          <p className="mb-1.5 text-center text-xs text-subtle">
            Can&apos;t scan? Enter this key manually
          </p>
          <button
            type="button"
            onClick={copySecret}
            className="flex w-full items-center justify-between gap-3 rounded-xl border border-line bg-white/5 px-4 py-3 font-mono text-sm tracking-wider text-foreground transition-colors hover:border-line-strong"
          >
            {secret || "…"}
            <span className="flex items-center gap-1 text-xs text-muted">
              {copied ? (
                <>
                  <Check className="h-3.5 w-3.5 text-emerald-400" /> Copied
                </>
              ) : (
                <>
                  <Copy className="h-3.5 w-3.5" /> Copy
                </>
              )}
            </span>
          </button>
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <Button
          size="lg"
          className="w-full"
          disabled={!secret}
          onClick={() => {
            setError(undefined);
            setStep("confirm");
          }}
        >
          I&apos;ve added it — continue
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
        <Link
          href="/dashboard"
          className="text-center text-sm text-muted transition-colors hover:text-accent"
        >
          Skip for now
        </Link>
      </div>
    </div>
  );
}
