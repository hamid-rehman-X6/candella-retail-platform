"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Smartphone, KeyRound, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { OtpInput } from "@/components/auth/otp-input";
import { TextField } from "@/components/auth/text-field";
import { Button } from "@/components/ui/button";

export function TwoFactorForm() {
  const router = useRouter();
  const [mode, setMode] = useState<"app" | "backup">("app");
  const [code, setCode] = useState("");
  const [backup, setBackup] = useState("");
  const [error, setError] = useState<string>();
  const [verifying, setVerifying] = useState(false);

  function verify(value: string) {
    setVerifying(true);
    setError(undefined);
    // TODO: verify the TOTP / backup code against the backend.
    setTimeout(() => {
      if (value === "000000") {
        setVerifying(false);
        setError("Invalid code. Open your authenticator app and try again.");
        setCode("");
        return;
      }
      router.push("/");
    }, 800);
  }

  function verifyBackup(e: React.FormEvent) {
    e.preventDefault();
    if (backup.trim().length < 8) {
      setError("Enter one of your backup codes.");
      return;
    }
    setVerifying(true);
    setError(undefined);
    setTimeout(() => router.push("/"), 800);
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        icon={mode === "app" ? Smartphone : KeyRound}
        title="Two-factor authentication"
        subtitle={
          mode === "app"
            ? "Enter the 6-digit code from your authenticator app to finish signing in."
            : "Enter one of the backup codes you saved when setting up 2FA."
        }
      />

      {mode === "app" ? (
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
                Verify
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </>
            )}
          </Button>

          <button
            type="button"
            onClick={() => {
              setMode("backup");
              setError(undefined);
            }}
            className="text-center text-sm text-muted transition-colors hover:text-accent"
          >
            Can&apos;t access your app? Use a backup code
          </button>
        </div>
      ) : (
        <form onSubmit={verifyBackup} className="flex flex-col gap-4">
          <TextField
            label="Backup code"
            icon={KeyRound}
            placeholder="XXXX-XXXX"
            value={backup}
            onChange={(e) => setBackup(e.target.value)}
            error={error}
          />
          <Button
            type="submit"
            size="lg"
            className="mt-2 w-full"
            disabled={verifying}
          >
            {verifying ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Verifying…
              </>
            ) : (
              "Verify backup code"
            )}
          </Button>

          <button
            type="button"
            onClick={() => {
              setMode("app");
              setError(undefined);
            }}
            className="text-center text-sm text-muted transition-colors hover:text-accent"
          >
            Use authenticator app instead
          </button>
        </form>
      )}

      <p className="text-center text-sm text-muted">
        <Link
          href="/login"
          className="font-medium text-accent transition-colors hover:text-accent-bright"
        >
          Back to sign in
        </Link>
      </p>
    </div>
  );
}
