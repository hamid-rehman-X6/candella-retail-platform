import type { CSSProperties } from "react";
import { cn } from "@/lib/cn";

type MarqueeProps = {
  children: React.ReactNode;
  reverse?: boolean;
  /** Seconds for one full loop. Lower = faster. */
  durationSeconds?: number;
  /** CSS length used both between items and between the two groups. */
  gap?: string;
  className?: string;
};

/**
 * Seamless, infinite auto-scrolling row. Renders its children twice (two
 * identical groups) and scrolls them so the row never shows a blank edge, at any
 * viewport width — see the `.marquee` rules in globals.css. Pass the row of items
 * as children (keep each item `shrink-0`); the second copy is aria-hidden.
 */
export function Marquee({
  children,
  reverse = false,
  durationSeconds = 42,
  gap = "3.5rem",
  className,
}: MarqueeProps) {
  const style = {
    "--marquee-gap": gap,
    "--marquee-duration": `${durationSeconds}s`,
  } as CSSProperties;

  const groupClass = cn("marquee__group", reverse && "marquee__group--reverse");

  return (
    <div className={cn("marquee mask-fade-edges", className)} style={style}>
      <div className={groupClass}>{children}</div>
      <div className={groupClass} aria-hidden>
        {children}
      </div>
    </div>
  );
}
