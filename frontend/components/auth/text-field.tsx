"use client";

import { useId } from "react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/cn";

type TextFieldProps = {
  label: string;
  icon?: LucideIcon;
  error?: string;
  hint?: string;
  className?: string;
} & React.InputHTMLAttributes<HTMLInputElement>;

export function TextField({
  label,
  icon: Icon,
  error,
  hint,
  className,
  id,
  type = "text",
  ...props
}: TextFieldProps) {
  const autoId = useId();
  const inputId = id ?? autoId;

  return (
    <div className={cn("flex flex-col gap-1.5", className)}>
      <label
        htmlFor={inputId}
        className="text-sm font-medium text-foreground/90"
      >
        {label}
      </label>
      <div className="relative">
        {Icon && (
          <Icon className="pointer-events-none absolute top-1/2 left-3.5 h-4 w-4 -translate-y-1/2 text-subtle" />
        )}
        <input
          id={inputId}
          type={type}
          aria-invalid={!!error}
          className={cn(
            "h-12 w-full rounded-xl border bg-white/5 px-3.5 text-sm text-foreground transition-colors outline-none placeholder:text-subtle focus:ring-2 focus:ring-accent/20",
            Icon && "pl-11",
            error
              ? "border-red-500/60 focus:border-red-500/60"
              : "border-line focus:border-accent/50",
          )}
          {...props}
        />
      </div>
      {error ? (
        <p className="text-xs text-red-400">{error}</p>
      ) : hint ? (
        <p className="text-xs text-subtle">{hint}</p>
      ) : null}
    </div>
  );
}
