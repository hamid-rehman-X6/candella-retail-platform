import Link from "next/link";
import { cn } from "@/lib/cn";

/** Candella brand mark — a luminous spark inside a rounded token. */
export function LogoMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "relative inline-flex h-9 w-9 items-center justify-center rounded-xl bg-linear-to-br from-accent-bright to-accent-deep shadow-[0_4px_20px_-4px_rgba(245,158,11,0.7)]",
        className,
      )}
    >
      <svg viewBox="0 0 24 24" fill="none" className="h-5 w-5" aria-hidden>
        <path
          d="M12 2.5c1.6 2.6 4.2 4.2 4.2 7.6a4.2 4.2 0 0 1-8.4 0c0-1.5.6-2.7 1.4-3.8.3 1 .9 1.7 1.7 2.1-.2-2.1.4-4 1.1-5.9Z"
          fill="#1a1206"
        />
        <circle cx="12" cy="17.5" r="3.2" fill="#1a1206" opacity="0.55" />
      </svg>
    </span>
  );
}

export function Logo({ className }: { className?: string }) {
  // Always routes to the home page. Using a real route (not a "#top" anchor) means
  // it works from the auth screens too, and never leaves a dangling #top in the URL.
  return (
    <Link
      href="/"
      className={cn("group inline-flex items-center gap-2.5", className)}
    >
      <LogoMark />
      <span className="text-lg font-semibold tracking-tight text-foreground">
        Candella
      </span>
    </Link>
  );
}
