/**
 * Human copy for the machine-readable auth error codes the backend emits.
 * Shared by the OAuth bridge page (as a toast) and the login form (as an inline
 * alert), so a given failure always reads the same way wherever it surfaces.
 */

export type AuthErrorCopy = { title: string; description: string };

const FALLBACK: AuthErrorCopy = {
  title: "We couldn't complete your sign-in",
  description: "Something went wrong on the way back from Google. Try again.",
};

const COPY: Record<string, AuthErrorCopy> = {
  google_not_configured: {
    title: "Google sign-in isn't available",
    description:
      "It hasn't been set up on this server yet. Sign in with your email and password instead.",
  },
  google_email_unverified: {
    title: "Your Google email isn't verified",
    description:
      "Verify your email address with Google, then try signing in again.",
  },
  google_link_blocked: {
    title: "This email already has an account",
    description:
      "An unverified account exists for this address. Sign in with your password first, then you can use Google.",
  },
  token_invalid: {
    title: "That sign-in attempt expired",
    description:
      "Sign-in links are valid for a few minutes. Please start again.",
  },
  google_failed: FALLBACK,
};

/** Resolves an error code to display copy, falling back to a generic message. */
export function authErrorCopy(code?: string | null): AuthErrorCopy {
  return (code && COPY[code]) || FALLBACK;
}
