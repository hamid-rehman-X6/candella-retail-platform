"use client";

import { useId, useState } from "react";
import { Lock, Eye, EyeOff } from "lucide-react";
import { cn } from "@/lib/cn";
import { PasswordStrengthMeter } from "./password-strength";

type PasswordFieldProps = {
  label: string;
  value: string;
  error?: string;
  showStrength?: boolean;
  className?: string;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, "type">;

export function PasswordField({
  label,
  value,
  error,
  showStrength = false,
  className,
  id,
  ...props
}: PasswordFieldProps) {
  const autoId = useId();
  const inputId = id ?? autoId;
  const [visible, setVisible] = useState(false);

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <label
        htmlFor={inputId}
        className="text-sm font-medium text-foreground/90"
      >
        {label}
      </label>
      <div className="relative">
        <Lock className="pointer-events-none absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2 text-subtle" />
        <input
          id={inputId}
          type={visible ? "text" : "password"}
          value={value}
          aria-invalid={!!error}
          className={cn(
            "h-12 w-full rounded-xl border bg-white/5 px-3.5 pr-11 pl-11 text-sm text-foreground transition-colors outline-none placeholder:text-subtle focus:ring-2 focus:ring-accent/20",
            error
              ? "border-red-500/60 focus:border-red-500/60"
              : "border-line focus:border-accent/50",
          )}
          {...props}
        />
        <button
          type="button"
          onClick={() => setVisible((v) => !v)}
          aria-label={visible ? "Hide password" : "Show password"}
          className="absolute top-1/2 right-3 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-lg text-subtle transition-colors hover:text-foreground"
        >
          {visible ? (
            <EyeOff className="h-4 w-4" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </button>
      </div>
      {error && <p className="text-xs text-red-400">{error}</p>}
      {showStrength && <PasswordStrengthMeter password={value} />}
    </div>
  );
}
