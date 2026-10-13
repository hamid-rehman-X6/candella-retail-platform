"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft, ArrowRight, CheckCircle2, Loader2 } from "lucide-react";
import { LogoMark } from "@/components/ui/logo";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/auth/text-field";
import { industries } from "@/lib/industries";
import { ApiError } from "@/lib/api";
import { createWorkspace } from "@/lib/auth-api";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/cn";

export function OnboardingForm({
  userName,
  signedInWith,
  returnHref,
  returnLabel,
}: {
  userName: string;
  /** Email to confirm in the banner, when arriving straight from Google. */
  signedInWith?: string;
  returnHref: string;
  returnLabel: string;
}) {
  const router = useRouter();
  const firstName = userName.split(" ")[0] || userName;
  const [name, setName] = useState("");
  const [industry, setIndustry] = useState("pos_general");
  const [error, setError] = useState<string>();
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (name.trim().length < 2) {
      setError("Enter a name for your business.");
      return;
    }
    setError(undefined);
    setLoading(true);
    try {
      const { workspace } = await createWorkspace({
        name,
        industryType: industry,
      });
      toast.success(`${workspace.name} is ready`, {
        description: "Your workspace has been created.",
      });
      router.push(`/w/${workspace.slug}`);
      router.refresh();
    } catch (err) {
      setLoading(false);
      setError(
        err instanceof ApiError
          ? err.message
          : "Couldn't create your workspace. Please try again.",
      );
    }
  }

  return (
    <div className="mx-auto w-full max-w-lg">
      <Link
        href={returnHref}
        className="mb-8 inline-flex items-center gap-2 rounded-lg px-2 py-1 text-sm text-muted transition-colors hover:text-foreground focus-visible:outline-2 focus-visible:outline-accent"
      >
        <ArrowLeft className="h-4 w-4" />
        {returnLabel}
      </Link>
      <div className="flex flex-col items-center text-center">
        <LogoMark className="h-12 w-12 rounded-2xl" />
        <h1 className="mt-6 text-2xl font-semibold tracking-tight sm:text-3xl">
          Let&apos;s set up your business
        </h1>
        <p className="mt-2 text-muted">
          Welcome, {firstName}. Create your workspace to get started — you can
          change any of this later.
        </p>
      </div>

      {signedInWith && (
        <div className="mt-6 flex items-center gap-3 rounded-xl border border-emerald-500/25 bg-emerald-500/10 px-4 py-3">
          <CheckCircle2 className="h-4.5 w-4.5 shrink-0 text-emerald-400" />
          <p className="min-w-0 text-sm text-emerald-200/90">
            Signed in with Google as{" "}
            <span className="font-medium break-all text-emerald-100">
              {signedInWith}
            </span>
          </p>
        </div>
      )}

      {error && (
        <p className="mt-6 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300">
          {error}
        </p>
      )}

      <form onSubmit={handleSubmit} className="mt-8 flex flex-col gap-6">
        <TextField
          label="Business name"
          placeholder="Acme Pharmacy"
          value={name}
          onChange={(e) => setName(e.target.value)}
          autoFocus
        />

        <div>
          <label className="text-sm font-medium text-foreground/90">
            What do you sell?
          </label>
          <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
            {industries.map((opt) => {
              const Icon = opt.icon;
              const selected = industry === opt.value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => setIndustry(opt.value)}
                  className={cn(
                    "flex items-center gap-2.5 rounded-xl border px-3 py-3 text-left text-sm transition-colors",
                    selected
                      ? "border-accent/50 bg-accent/10 text-foreground"
                      : "border-line bg-white/[0.02] text-muted hover:border-line-strong hover:text-foreground",
                  )}
                >
                  <Icon
                    className={cn(
                      "h-4.5 w-4.5 shrink-0",
                      selected ? "text-accent" : "text-subtle",
                    )}
                  />
                  <span className="truncate">{opt.label}</span>
                </button>
              );
            })}
          </div>
        </div>

        <Button type="submit" size="lg" className="w-full" disabled={loading}>
          {loading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" />
              Creating workspace…
            </>
          ) : (
            <>
              Create workspace
              <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </>
          )}
        </Button>
      </form>
    </div>
  );
}
