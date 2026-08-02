"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { CircleCheck, TriangleAlert, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { PasswordField } from "@/components/auth/password-field";
import { Button } from "@/components/ui/button";
import { getPasswordStrength } from "@/lib/validation";
import { ApiError } from "@/lib/api";
import { resetPassword } from "@/lib/auth-api";

export function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [errors, setErrors] = useState<{ password?: string; confirm?: string }>(
    {},
  );
  const [formError, setFormError] = useState<string>();
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const next: typeof errors = {};
    if (getPasswordStrength(password).score < 2)
      next.password = "Choose a stronger password (8+ chars, mixed types).";
    if (confirm !== password) next.confirm = "Passwords don't match.";
    setErrors(next);
    if (Object.keys(next).length > 0) return;

    setFormError(undefined);
    setLoading(true);
    try {
      await resetPassword({ token, password });
      setDone(true);
    } catch (err) {
      setLoading(false);
      if (err instanceof ApiError && err.code === "weak_password") {
        setErrors({ password: err.message });
      } else {
        setFormError(
          err instanceof ApiError
            ? err.message
            : "Something went wrong. Please try again.",
        );
      }
    }
  }

  // No token in the URL — the link is missing or malformed.
  if (!token) {
    return (
      <div className="flex flex-col gap-8">
        <AuthHeader
          icon={TriangleAlert}
          title="Invalid reset link"
          subtitle="This password reset link is missing or malformed. Request a new one to continue."
        />
        <Button href="/forgot-password" size="lg" className="w-full">
          Request a new link
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
      </div>
    );
  }

  if (done) {
    return (
      <div className="flex flex-col gap-8">
        <AuthHeader
          icon={CircleCheck}
          title="Password updated"
          subtitle="Your password has been changed. You can now sign in with your new password."
        />
        <Button
          size="lg"
          className="w-full"
          onClick={() => router.push("/login")}
        >
          Continue to sign in
          <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        title="Set a new password"
        subtitle="Choose a strong password you don't use anywhere else."
      />

      {formError && (
        <p className="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
          {formError}
        </p>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <PasswordField
          label="New password"
          placeholder="Create a new password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={errors.password}
          showStrength
        />
        <PasswordField
          label="Confirm password"
          placeholder="Re-enter your new password"
          autoComplete="new-password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          error={errors.confirm}
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
              Updating…
            </>
          ) : (
            <>
              Reset password
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>
      </form>

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
