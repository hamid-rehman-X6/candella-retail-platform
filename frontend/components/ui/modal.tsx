"use client";

import { useEffect, useId, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/lib/cn";

type ModalProps = {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  description?: ReactNode;
  icon?: ReactNode;
  /** Buttons pinned below the scrolling body, e.g. Cancel and Save. */
  footer?: ReactNode;
  children?: ReactNode;
  /** While true, Escape, the backdrop and the close button do nothing. */
  busy?: boolean;
  size?: "sm" | "md" | "lg";
  role?: "dialog" | "alertdialog";
};

const widths = { sm: "max-w-sm", md: "max-w-md", lg: "max-w-lg" };

/**
 * The one dialog every screen uses. It renders into document.body, so no
 * transformed or blurred ancestor can stop the backdrop covering the screen.
 * The header and footer stay put; only the body scrolls.
 */
export function Modal({
  open,
  onClose,
  title,
  description,
  icon,
  footer,
  children,
  busy = false,
  size = "md",
  role = "dialog",
}: ModalProps) {
  const titleId = useId();
  const descriptionId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const closeRef = useRef(onClose);
  const busyRef = useRef(busy);

  useEffect(() => {
    closeRef.current = onClose;
    busyRef.current = busy;
  });

  useEffect(() => {
    if (!open) return;
    const trigger = document.activeElement as HTMLElement | null;
    const body = document.body;
    const scrollbar = window.innerWidth - document.documentElement.clientWidth;
    const previous = {
      overflow: body.style.overflow,
      paddingRight: body.style.paddingRight,
    };
    // Lock the page behind the dialog without letting it jump sideways.
    body.style.overflow = "hidden";
    if (scrollbar > 0) body.style.paddingRight = `${scrollbar}px`;

    const panel = panelRef.current;
    const first = panel?.querySelector<HTMLElement>(
      "[autofocus], input:not([type=hidden]):not(:disabled), textarea:not(:disabled), select:not(:disabled), button:not(:disabled):not([data-modal-close])",
    );
    (first ?? panel)?.focus();

    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "Escape" || busyRef.current) return;
      // A dropdown open inside the dialog handles its own Escape first.
      if (event.defaultPrevented) return;
      event.preventDefault();
      closeRef.current();
    }
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      body.style.overflow = previous.overflow;
      body.style.paddingRight = previous.paddingRight;
      trigger?.focus?.();
    };
  }, [open]);

  if (!open || typeof document === "undefined") return null;

  return createPortal(
    <div
      className="fixed inset-0 z-[90] flex items-end justify-center bg-black/50 p-0 backdrop-blur-[2px] sm:items-center sm:p-4"
      role="presentation"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
    >
      <div
        ref={panelRef}
        role={role}
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        tabIndex={-1}
        className={cn(
          "flex max-h-[92dvh] w-full flex-col overflow-hidden rounded-t-2xl border border-line bg-canvas-raised shadow-2xl outline-none sm:max-h-[88dvh] sm:rounded-2xl",
          widths[size],
        )}
      >
        <div className="flex items-start justify-between gap-4 border-b border-line px-5 py-4 sm:px-6">
          <div className="flex min-w-0 items-start gap-3">
            {icon && <span className="mt-0.5 text-accent-ink">{icon}</span>}
            <div className="min-w-0">
              <h2 id={titleId} className="text-lg font-semibold">
                {title}
              </h2>
              {description && (
                <p id={descriptionId} className="mt-0.5 text-sm text-muted">
                  {description}
                </p>
              )}
            </div>
          </div>
          <button
            type="button"
            data-modal-close
            onClick={onClose}
            disabled={busy}
            aria-label="Close"
            className="-mr-1 shrink-0 cursor-pointer rounded-lg p-1.5 text-muted transition-colors hover:bg-surface hover:text-foreground disabled:opacity-40"
          >
            <X size={18} />
          </button>
        </div>
        {children && (
          <div className="scroll-thin min-h-0 flex-1 overflow-y-auto overscroll-contain px-5 py-5 sm:px-6">
            {children}
          </div>
        )}
        {footer && (
          <div className="flex flex-wrap justify-end gap-3 border-t border-line bg-canvas-raised px-5 py-4 sm:px-6">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body,
  );
}

export const modalButton = {
  primary:
    "inline-flex min-h-10 cursor-pointer items-center justify-center rounded-xl bg-accent px-4 py-2.5 text-sm font-semibold text-black transition-colors hover:bg-accent-bright disabled:cursor-not-allowed disabled:opacity-50",
  secondary:
    "inline-flex min-h-10 cursor-pointer items-center justify-center rounded-xl border border-line px-4 py-2.5 text-sm font-medium transition-colors hover:border-line-strong disabled:cursor-not-allowed disabled:opacity-50",
  danger:
    "inline-flex min-h-10 cursor-pointer items-center justify-center rounded-xl bg-red-600 px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-50",
};

/** A yes/no question asked in-app instead of the browser's confirm(). */
export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel,
  tone = "primary",
  busy = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: ReactNode;
  message: ReactNode;
  confirmLabel: string;
  tone?: "primary" | "danger";
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <Modal
      open={open}
      onClose={onCancel}
      title={title}
      description={message}
      busy={busy}
      size="sm"
      role="alertdialog"
      footer={
        <>
          <button
            type="button"
            autoFocus
            disabled={busy}
            onClick={onCancel}
            className={modalButton.secondary}
          >
            Cancel
          </button>
          <button
            type="button"
            disabled={busy}
            onClick={onConfirm}
            className={
              tone === "danger" ? modalButton.danger : modalButton.primary
            }
          >
            {busy ? "Working…" : confirmLabel}
          </button>
        </>
      }
    />
  );
}
