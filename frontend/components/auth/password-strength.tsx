"use client";

import { getPasswordStrength } from "@/lib/validation";
import { cn } from "@/lib/cn";

const barColor = [
  "bg-white/10",
  "bg-red-500",
  "bg-orange-500",
  "bg-yellow-500",
  "bg-emerald-500",
];

export function PasswordStrengthMeter({ password }: { password: string }) {
  const { score, label } = getPasswordStrength(password);
  if (!password) return null;

  return (
    <div className="mt-1.5">
      <div className="flex gap-1.5">
        {[1, 2, 3, 4].map((i) => (
          <span
            key={i}
            className={cn(
              "h-1.5 flex-1 rounded-full transition-colors duration-300",
              i <= score ? barColor[score] : "bg-white/10",
            )}
          />
        ))}
      </div>
      <p className="mt-1.5 text-xs text-subtle">
        Password strength:{" "}
        <span className="font-medium text-foreground/80">{label}</span>
      </p>
    </div>
  );
}
