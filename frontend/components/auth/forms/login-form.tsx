"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Mail, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { TextField } from "@/components/auth/text-field";
import { PasswordField } from "@/components/auth/password-field";
import { GoogleButton } from "@/components/auth/google-button";
import { Divider } from "@/components/auth/divider";
import { Button } from "@/components/ui/button";
import { isValidEmail } from "@/lib/validation";
import { ApiError } from "@/lib/api";
import { login, googleStartUrl, pendingEmail, mfaToken } from "@/lib/auth-api";

export function LoginForm() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [errors, setErrors] = useState<{ email?: string; password?: string }>(
    {},
  );
  const [formError, setFormError] = useState<string>();
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const next: typeof errors = {};
    if (!isValidEmail(email)) next.email = "Enter a valid email address.";
    if (password.length < 1) next.password = "Enter your password.";
    setErrors(next);
    if (Object.keys(next).length > 0) return;

    setFormError(undefined);
    setLoading(true);
    try {
      const res = await login({ email, password });
      if (res.mfaRequired && res.mfaToken) {
        mfaToken.set(res.mfaToken);
        router.push("/two-factor");
        return;
      }
      // Signed in. TODO: route to /dashboard once it exists.
      router.push("/");
    } catch (err) {
      setLoading(false);
      if (err instanceof ApiError) {
        if (err.code === "email_not_verified") {
          pendingEmail.set(email);
          router.push("/verify-email");
          return;
        }
        setFormError(err.message);
      } else {
        setFormError("Something went wrong. Please try again.");
      }
    }
  }

  function handleGoogle() {
    window.location.href = googleStartUrl;
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        title="Welcome back"
        subtitle="Sign in to your Candella workspace to keep every counter running."
      />

      <GoogleButton onClick={handleGoogle} disabled={loading} />
      <Divider />

      {formError && (
        <p className="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
          {formError}
        </p>
      )}

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Email"
          type="email"
          icon={Mail}
          placeholder="you@company.com"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={errors.email}
        />

        <div className="flex flex-col gap-1.5">
          <PasswordField
            label="Password"
            placeholder="Enter your password"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={errors.password}
          />
          <div className="flex justify-end">
            <Link
              href="/forgot-password"
              className="text-xs text-muted transition-colors hover:text-accent"
            >
              Forgot password?
            </Link>
          </div>
        </div>

        <Button
          type="submit"
          size="lg"
          className="mt-2 w-full"
          disabled={loading}
        >
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Signing in…
            </>
          ) : (
            <>
              Sign in
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        Don&apos;t have an account?{" "}
        <Link
          href="/signup"
          className="font-medium text-accent transition-colors hover:text-accent-bright"
        >
          Create one
        </Link>
      </p>
    </div>
  );
}
