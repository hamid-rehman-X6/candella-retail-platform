"use client";

import { useSyncExternalStore } from "react";

/**
 * The app's global feedback channel.
 *
 * It lives in a module-level store rather than React context for one reason:
 * redirect-based flows (Google sign-in) and plain async helpers need to raise a
 * message from outside a component tree. `toast.success(...)` works anywhere.
 *
 * Render <Toaster /> once in the root layout to display them.
 */

export type ToastVariant = "success" | "error" | "info";

export type Toast = {
  id: number;
  variant: ToastVariant;
  title: string;
  description?: string;
};

type ToastOptions = {
  description?: string;
  /** Milliseconds before auto-dismiss. Defaults to 5s, or 8s for errors. */
  duration?: number;
};

/** Older toasts are dropped once this many are on screen. */
const MAX_VISIBLE = 3;

type Timer = {
  handle: ReturnType<typeof setTimeout>;
  expiresAt: number;
  remaining: number;
};

let toasts: Toast[] = [];
let nextId = 1;
let paused = false;
const timers = new Map<number, Timer>();
const listeners = new Set<() => void>();

const EMPTY: Toast[] = [];

function publish(next: Toast[]) {
  toasts = next;
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

function clearTimer(id: number) {
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer.handle);
    timers.delete(id);
  }
}

function startTimer(id: number, ms: number) {
  timers.set(id, {
    handle: setTimeout(() => dismissToast(id), ms),
    expiresAt: Date.now() + ms,
    remaining: ms,
  });
}

function push(variant: ToastVariant, title: string, options?: ToastOptions) {
  const id = nextId++;
  const duration = options?.duration ?? (variant === "error" ? 8000 : 5000);

  const next = [
    ...toasts,
    { id, variant, title, description: options?.description },
  ];
  // Drop the oldest beyond the cap, cancelling their timers as we go.
  while (next.length > MAX_VISIBLE) {
    const oldest = next.shift();
    if (oldest) clearTimer(oldest.id);
  }

  publish(next);
  startTimer(id, duration);
  return id;
}

export function dismissToast(id: number) {
  clearTimer(id);
  publish(toasts.filter((t) => t.id !== id));
}

/** Freezes auto-dismiss while the pointer rests on the stack. */
export function pauseToasts() {
  if (paused) return;
  paused = true;
  for (const [id, timer] of timers) {
    clearTimeout(timer.handle);
    timers.set(id, {
      ...timer,
      remaining: Math.max(0, timer.expiresAt - Date.now()),
    });
  }
}

/** Resumes each toast with the time it had left when it was paused. */
export function resumeToasts() {
  if (!paused) return;
  paused = false;
  for (const [id, timer] of timers) {
    clearTimeout(timer.handle);
    startTimer(id, timer.remaining);
  }
}

export const toast = {
  success: (title: string, options?: ToastOptions) =>
    push("success", title, options),
  error: (title: string, options?: ToastOptions) =>
    push("error", title, options),
  info: (title: string, options?: ToastOptions) => push("info", title, options),
};

/** Subscribes a component to the live toast list. */
export function useToasts(): Toast[] {
  return useSyncExternalStore(
    subscribe,
    () => toasts,
    () => EMPTY,
  );
}
