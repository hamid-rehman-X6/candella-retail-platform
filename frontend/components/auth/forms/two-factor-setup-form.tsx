"use client";

import { useState } from "react";
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

// Demo values only — a real secret + backup codes come from the backend at setup.
const DEMO_SECRET = "JBSWY3DPEHPK3PXP";
const OTPAUTH_URL = `otpauth://totp/Candella:you@company.com?secret=${DEMO_SECRET}&issuer=Candella&algorithm=SHA1&digits=6&period=30`;
const BACKUP_CODES = [
  "4F9K-2QMX",
  "8ZC3-RTP7",
  "1WN6-LK9D",
  "5HB2-XQ4V",
  "9TM8-PC3R",
  "2KD7-VN6L",
  "6RX4-ZQ8M",
  "3PL9-WH2K",
];

type Step = "scan" | "confirm" | "done";

export function TwoFactorSetupForm() {
  const router = useRouter();
  const [step, setStep] = useState<Step>("scan");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string>();
  const [verifying, setVerifying] = useState(false);
  const [copied, setCopied] = useState(false);

  async function copySecret() {
    try {
      await navigator.clipboard.writeText(DEMO_SECRET);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* clipboard unavailable */
    }
  }

  function confirm(value: string) {
    setVerifying(true);
    setError(undefined);
    // TODO: verify the first TOTP code against the backend to enable 2FA.
    setTimeout(() => {
      setVerifying(false);
      if (value === "000000") {
        setError("That code didn't match. Wait for the next one and retry.");
        setCode("");
        return;
      }
      setStep("done");
    }, 800);
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
          {BACKUP_CODES.map((c) => (
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

        <Button size="lg" className="w-full" onClick={() => router.push("/")}>
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

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        icon={ShieldCheck}
        title="Set up two-factor auth"
        subtitle="Scan this QR code with an authenticator app like Google Authenticator, 1Password or Authy."
      />

      <div className="flex flex-col items-center gap-5">
        <div className="rounded-2xl bg-white p-4 shadow-lg">
          <QRCodeSVG value={OTPAUTH_URL} size={168} level="M" />
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
            {DEMO_SECRET}
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
        <Button size="lg" className="w-full" onClick={() => setStep("confirm")}>
          I&apos;ve added it — continue
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
        <Link
          href="/"
          className="text-center text-sm text-muted transition-colors hover:text-accent"
        >
          Skip for now
        </Link>
      </div>
    </div>
  );
}
