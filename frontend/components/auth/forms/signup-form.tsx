"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Mail, User, ArrowRight, Loader2 } from "lucide-react";
import { AuthHeader } from "@/components/auth/auth-header";
import { TextField } from "@/components/auth/text-field";
import { PasswordField } from "@/components/auth/password-field";
import { GoogleButton } from "@/components/auth/google-button";
import { Divider } from "@/components/auth/divider";
import { Button } from "@/components/ui/button";
import { isValidEmail, getPasswordStrength } from "@/lib/validation";

export function SignupForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [agreed, setAgreed] = useState(false);
  const [errors, setErrors] = useState<{
    name?: string;
    email?: string;
    password?: string;
    agreed?: string;
  }>({});
  const [loading, setLoading] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const next: typeof errors = {};
    if (name.trim().length < 2) next.name = "Enter your full name.";
    if (!isValidEmail(email)) next.email = "Enter a valid email address.";
    if (getPasswordStrength(password).score < 2)
      next.password = "Choose a stronger password (8+ chars, mixed types).";
    if (!agreed) next.agreed = "Please accept the terms to continue.";
    setErrors(next);
    if (Object.keys(next).length > 0) return;

    setLoading(true);
    // TODO: replace with real sign-up call; backend then emails a 6-digit code.
    setTimeout(() => router.push("/verify-email"), 900);
  }

  function handleGoogle() {
    setGoogleLoading(true);
    setTimeout(() => router.push("/verify-email"), 900);
  }

  return (
    <div className="flex flex-col gap-8">
      <AuthHeader
        title="Create your account"
        subtitle="Start your 14-day free trial. No credit card required."
      />

      <GoogleButton
        label="Sign up with Google"
        onClick={handleGoogle}
        disabled={googleLoading || loading}
      />
      <Divider />

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Full name"
          icon={User}
          placeholder="Ayesha Khan"
          autoComplete="name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          error={errors.name}
        />
        <TextField
          label="Work email"
          type="email"
          icon={Mail}
          placeholder="you@company.com"
          autoComplete="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          error={errors.email}
        />
        <PasswordField
          label="Password"
          placeholder="Create a password"
          autoComplete="new-password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          error={errors.password}
          showStrength
        />

        <label className="flex items-start gap-2.5 text-sm text-muted">
          <input
            type="checkbox"
            checked={agreed}
            onChange={(e) => setAgreed(e.target.checked)}
            className="mt-0.5 h-4 w-4 shrink-0 rounded border-line bg-white/5 accent-accent"
          />
          <span>
            I agree to the{" "}
            <Link href="#" className="text-accent hover:text-accent-bright">
              Terms
            </Link>{" "}
            and{" "}
            <Link href="#" className="text-accent hover:text-accent-bright">
              Privacy Policy
            </Link>
            .
          </span>
        </label>
        {errors.agreed && (
          <p className="-mt-2 text-xs text-red-400">{errors.agreed}</p>
        )}

        <Button
          type="submit"
          size="lg"
          className="mt-2 w-full"
          disabled={loading}
        >
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Creating account…
            </>
          ) : (
            <>
              Create account
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-muted">
        Already have an account?{" "}
        <Link
          href="/login"
          className="font-medium text-accent transition-colors hover:text-accent-bright"
        >
          Sign in
        </Link>
      </p>
    </div>
  );
}
