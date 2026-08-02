"use client";

import { useState } from "react";
import Link from "next/link";
import { Mail, MailCheck, ArrowLeft, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { TextField } from "@/components/auth/text-field";
import { Button } from "@/components/ui/button";
import { isValidEmail } from "@/lib/validation";
import { forgotPassword } from "@/lib/auth-api";

export function ForgotPasswordForm() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!isValidEmail(email)) {
      setError("Enter a valid email address.");
      return;
    }
    setError(undefined);
    setLoading(true);
    try {
      // The backend always responds the same way (no account enumeration).
      await forgotPassword(email);
      setSent(true);
    } catch {
      setError("Couldn't send the reset link. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  if (sent) {
    return (
      <div className="flex flex-col gap-8">
        <AuthHeader
          icon={MailCheck}
          title="Check your email"
          subtitle={
            <>
              If an account exists for{" "}
              <span className="font-medium text-foreground/90">{email}</span>,
              we&apos;ve sent a link to reset your password. The link expires in
              30 minutes.
            </>
          }
        />

        <button
          type="button"
          onClick={() => setSent(false)}
          className="text-center text-sm text-muted transition-colors hover:text-accent"
        >
          Didn&apos;t get it? Try another email
        </button>

        <p className="text-center text-sm text-muted">
          <Link
            href="/login"
            className="inline-flex items-center gap-1.5 font-medium text-accent transition-colors hover:text-accent-bright"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to sign in
          </Link>
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        title="Forgot your password?"
        subtitle="Enter the email tied to your account and we'll send you a link to reset it."
      />

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Email"
          type="email"
          icon={Mail}
          placeholder="you@company.com"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={error}
        />

        <Button
          type="submit"
          size="lg"
          className="mt-2 w-full"
          disabled={loading}
        >
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Sending link…
            </>
          ) : (
            <>
              Send reset link
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        <Link
          href="/login"
          className="inline-flex items-center gap-1.5 font-medium text-accent transition-colors hover:text-accent-bright"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to sign in
        </Link>
      </p>
    </div>
  );
}
