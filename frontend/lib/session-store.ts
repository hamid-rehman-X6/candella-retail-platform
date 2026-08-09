"use client";

import { useEffect, useSyncExternalStore } from "react";
import { getMe, type AuthUser } from "@/lib/auth-api";

/**
 * A tiny client-side session store for the *public* pages (landing, marketing).
 *
 * Those pages are statically rendered, so we can't read the httpOnly session
 * cookie on the server without turning them dynamic. Instead every consumer
 * shares this module-level store: the first component to mount fires a single
 * `GET /auth/me`, and every other component reads the same result — no
 * duplicate requests, no prop drilling.
 *
 * The authenticated app (`/dashboard`) does NOT use this: it's server-guarded
 * and seeds `AppProvider` from the server, so there's no fetch on first paint.
 */

export type SessionState =
  | { status: "loading"; user: null }
  | { status: "guest"; user: null }
  | { status: "authenticated"; user: AuthUser };

const LOADING: SessionState = { status: "loading", user: null };
const GUEST: SessionState = { status: "guest", user: null };

let state: SessionState = LOADING;
let inflight: Promise<void> | null = null;
const listeners = new Set<() => void>();

function publish(next: SessionState) {
  state = next;
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

const getSnapshot = () => state;
// Static HTML always renders the neutral "loading" branch, so hydration matches.
const getServerSnapshot = () => LOADING;

function load(): Promise<void> {
  inflight ??= getMe()
    .then(({ user }) => publish({ status: "authenticated", user }))
    .catch(() => publish(GUEST));
  return inflight;
}

/** Primes the store after a successful sign-in, avoiding a redundant request. */
export function setSessionUser(user: AuthUser) {
  inflight = Promise.resolve();
  publish({ status: "authenticated", user });
}

/**
 * Drops the cached session after signing out. `inflight` is reset so the next
 * public page re-checks with the backend — self-correcting if logout failed.
 */
export function clearSession() {
  inflight = null;
  publish(GUEST);
}

/** Subscribes to the shared session, kicking off the lookup on first use. */
export function useSession(): SessionState {
  const snapshot = useSyncExternalStore(
    subscribe,
    getSnapshot,
    getServerSnapshot,
  );
  useEffect(() => {
    void load();
  }, []);
  return snapshot;
}
