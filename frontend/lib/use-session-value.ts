import { useSyncExternalStore } from "react";

const noopSubscribe = () => () => {};

/**
 * Reads a client-only value (e.g. from sessionStorage) in a hydration-safe way.
 * The server snapshot is always an empty string, so SSR and the first client
 * render agree; React then swaps in the real client value without a mismatch —
 * and without a setState-in-effect.
 */
export function useSessionValue(read: () => string): string {
  return useSyncExternalStore(noopSubscribe, read, () => "");
}
