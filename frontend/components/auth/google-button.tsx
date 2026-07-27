"use client";

function GoogleIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5" aria-hidden>
      <path
        fill="#EA4335"
        d="M12 10.2v3.9h5.5c-.24 1.4-.96 2.6-2.05 3.4v2.8h3.3c1.94-1.8 3.05-4.4 3.05-7.5 0-.7-.06-1.36-.18-2z"
        transform="translate(0 0)"
      />
      <path
        fill="#4285F4"
        d="M12 22c2.7 0 4.97-.9 6.63-2.43l-3.3-2.56c-.9.6-2.05.96-3.33.96-2.56 0-4.73-1.73-5.5-4.06H3.1v2.55C4.75 19.72 8.1 22 12 22z"
      />
      <path
        fill="#FBBC05"
        d="M6.5 13.9c-.2-.6-.32-1.24-.32-1.9s.12-1.3.32-1.9V7.55H3.1A9.98 9.98 0 0 0 2 12c0 1.6.38 3.12 1.1 4.45z"
      />
      <path
        fill="#34A853"
        d="M12 6.04c1.47 0 2.78.5 3.82 1.5l2.85-2.85C16.97 3.06 14.7 2 12 2 8.1 2 4.75 4.28 3.1 7.55L6.5 10.1C7.27 7.77 9.44 6.04 12 6.04z"
      />
    </svg>
  );
}

export function GoogleButton({
  onClick,
  label = "Continue with Google",
  disabled,
}: {
  onClick?: () => void;
  label?: string;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="inline-flex h-12 w-full items-center justify-center gap-3 rounded-full border border-line bg-white/5 text-sm font-medium text-foreground transition-all hover:border-line-strong hover:bg-white/10 disabled:pointer-events-none disabled:opacity-50"
    >
      <GoogleIcon />
      {label}
    </button>
  );
}
