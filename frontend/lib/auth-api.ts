import { apiGet, apiPost, API_BASE_URL } from "@/lib/api";

/** The current user shape returned by the backend. */
export type AuthUser = {
  id: string;
  email: string;
  fullName: string;
  emailVerified: boolean;
  mfaEnabled: boolean;
};

// ---- Registration & email verification ------------------------------------

export function register(input: {
  fullName: string;
  email: string;
  password: string;
}) {
  return apiPost<{ userId: string; email: string }>("/auth/register", input);
}

export function verifyEmail(input: { email: string; code: string }) {
  return apiPost<{ verified: boolean; user: AuthUser }>(
    "/auth/verify-email",
    input,
  );
}

export function resendVerification(email: string) {
  return apiPost<{ sent: boolean }>("/auth/resend-verification", { email });
}

// ---- Login (with optional 2FA second step) --------------------------------

export type LoginResult = {
  mfaRequired: boolean;
  mfaToken?: string;
  user?: AuthUser;
};

export function login(input: { email: string; password: string }) {
  return apiPost<LoginResult>("/auth/login", input);
}

export function verifyTwoFactor(input: { mfaToken: string; code: string }) {
  return apiPost<{ ok: boolean; user: AuthUser }>(
    "/auth/login/verify-2fa",
    input,
  );
}

export function verifyBackupCode(input: {
  mfaToken: string;
  backupCode: string;
}) {
  return apiPost<{ ok: boolean; user: AuthUser }>(
    "/auth/login/verify-backup",
    input,
  );
}

// ---- 2FA management (authenticated) ---------------------------------------

export function setupTwoFactor() {
  return apiPost<{ secret: string; otpauthUrl: string }>("/auth/2fa/setup");
}

export function enableTwoFactor(code: string) {
  return apiPost<{ backupCodes: string[] }>("/auth/2fa/enable", { code });
}

export function disableTwoFactor(password: string) {
  return apiPost<{ disabled: boolean }>("/auth/2fa/disable", { password });
}

// ---- Password reset --------------------------------------------------------

export function forgotPassword(email: string) {
  return apiPost<{ sent: boolean }>("/auth/forgot-password", { email });
}

export function resetPassword(input: { token: string; password: string }) {
  return apiPost<{ reset: boolean }>("/auth/reset-password", input);
}

// ---- Session ---------------------------------------------------------------

export function logout() {
  return apiPost<{ ok: boolean }>("/auth/logout");
}

export function getMe() {
  return apiGet<{ user: AuthUser }>("/auth/me");
}

/** Full URL to kick off the Google OAuth redirect flow. */
export const googleStartUrl = `${API_BASE_URL}/api/v1/auth/google/start`;

// ---- Workspaces ------------------------------------------------------------

export type Workspace = {
  id: string;
  name: string;
  slug: string;
  industryType: string;
  status: string;
  currencyCode: string;
  timezone: string;
  role: string;
  createdAt: string;
};

export function getWorkspaces() {
  return apiGet<{ workspaces: Workspace[] }>("/workspaces");
}

export function createWorkspace(input: {
  name: string;
  industryType: string;
  currencyCode?: string;
  timezone?: string;
}) {
  return apiPost<{ workspace: Workspace }>("/workspaces", input);
}

// ---- Small helpers to carry state between auth steps (client-only) ---------

const EMAIL_KEY = "candella:pending-email";
const MFA_TOKEN_KEY = "candella:mfa-token";

export const pendingEmail = {
  set: (email: string) => sessionStorage.setItem(EMAIL_KEY, email),
  get: () =>
    typeof window === "undefined"
      ? ""
      : (sessionStorage.getItem(EMAIL_KEY) ?? ""),
  clear: () => sessionStorage.removeItem(EMAIL_KEY),
};

export const mfaToken = {
  set: (token: string) => sessionStorage.setItem(MFA_TOKEN_KEY, token),
  get: () =>
    typeof window === "undefined"
      ? ""
      : (sessionStorage.getItem(MFA_TOKEN_KEY) ?? ""),
  clear: () => sessionStorage.removeItem(MFA_TOKEN_KEY),
};
