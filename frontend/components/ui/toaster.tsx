"use client";

import { AnimatePresence, motion, useReducedMotion } from "motion/react";
import { CheckCircle2, AlertCircle, Info, X } from "lucide-react";
import {
  useToasts,
  dismissToast,
  pauseToasts,
  resumeToasts,
  type ToastVariant,
} from "@/lib/toast";

const styles: Record<
  ToastVariant,
  { icon: typeof CheckCircle2; tone: string; ring: string }
> = {
  success: {
    icon: CheckCircle2,
    tone: "text-success-ink",
    ring: "ring-emerald-400/20",
  },
  error: {
    icon: AlertCircle,
    tone: "text-danger-ink",
    ring: "ring-red-400/20",
  },
  info: { icon: Info, tone: "text-accent-ink", ring: "ring-accent/20" },
};

/**
 * Renders the global toast stack. Mounted once in the root layout so messages
 * survive client-side navigation — which is what lets the OAuth bridge page fire
 * a toast and immediately route away.
 *
 * Sits above the scroll-to-top button (bottom-6) rather than beside it.
 */
export function Toaster() {
  const toasts = useToasts();
  const reduce = useReducedMotion();

  return (
    <ol
      aria-live="polite"
      aria-label="Notifications"
      onPointerEnter={pauseToasts}
      onPointerLeave={resumeToasts}
      className="pointer-events-none fixed inset-x-4 bottom-24 z-100 flex flex-col items-center gap-2 sm:inset-x-auto sm:right-6 sm:items-end"
    >
      <AnimatePresence initial={false}>
        {toasts.map((t) => {
          const { icon: Icon, tone, ring } = styles[t.variant];
          return (
            <motion.li
              key={t.id}
              layout={!reduce}
              // Errors announce assertively; the rest ride the polite region.
              role={t.variant === "error" ? "alert" : undefined}
              initial={
                reduce ? { opacity: 0 } : { opacity: 0, y: 16, scale: 0.96 }
              }
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={reduce ? { opacity: 0 } : { opacity: 0, scale: 0.96 }}
              transition={{ duration: 0.25, ease: [0.22, 1, 0.36, 1] }}
              className={`glass pointer-events-auto flex w-full max-w-sm items-start gap-3 rounded-2xl p-4 shadow-[0_12px_40px_-12px_rgba(0,0,0,0.8)] ring-1 ${ring}`}
            >
              <Icon className={`mt-px h-5 w-5 shrink-0 ${tone}`} />
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium text-foreground">{t.title}</p>
                {t.description && (
                  <p className="mt-0.5 text-xs leading-relaxed break-words text-muted">
                    {t.description}
                  </p>
                )}
              </div>
              <button
                type="button"
                onClick={() => dismissToast(t.id)}
                aria-label="Dismiss notification"
                className="-m-1 shrink-0 rounded-lg p-1 text-subtle transition-colors hover:bg-foreground/5 hover:text-foreground"
              >
                <X className="h-4 w-4" />
              </button>
            </motion.li>
          );
        })}
      </AnimatePresence>
    </ol>
  );
}
