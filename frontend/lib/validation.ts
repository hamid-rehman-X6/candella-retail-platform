/**
 * Lightweight client-side validation helpers for the auth screens.
 * When the real backend lands, these can be replaced by / paired with
 * shared Zod schemas — the function signatures are intentionally simple.
 */

export function isValidEmail(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
}

export type PasswordStrength = {
  score: 0 | 1 | 2 | 3 | 4;
  label: "Too weak" | "Weak" | "Fair" | "Good" | "Strong";
  checks: {
    length: boolean;
    lower: boolean;
    upper: boolean;
    number: boolean;
    symbol: boolean;
  };
};

export function getPasswordStrength(password: string): PasswordStrength {
  const checks = {
    length: password.length >= 8,
    lower: /[a-z]/.test(password),
    upper: /[A-Z]/.test(password),
    number: /[0-9]/.test(password),
    symbol: /[^A-Za-z0-9]/.test(password),
  };

  const passed = Object.values(checks).filter(Boolean).length;

  // Map the number of satisfied rules to a 0–4 score.
  let score: PasswordStrength["score"] = 0;
  if (password.length === 0) score = 0;
  else if (passed <= 2) score = 1;
  else if (passed === 3) score = 2;
  else if (passed === 4) score = 3;
  else score = 4;

  const labels = ["Too weak", "Weak", "Fair", "Good", "Strong"] as const;

  return { score, label: labels[score], checks };
}
