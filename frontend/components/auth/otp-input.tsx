"use client";

import { useRef } from "react";
import { cn } from "@/lib/cn";

type OtpInputProps = {
  length?: number;
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  disabled?: boolean;
  autoFocus?: boolean;
  error?: boolean;
};

export function OtpInput({
  length = 6,
  value,
  onChange,
  onComplete,
  disabled,
  autoFocus,
  error,
}: OtpInputProps) {
  const refs = useRef<Array<HTMLInputElement | null>>([]);
  const digits = Array.from({ length }, (_, i) => value[i] ?? "");

  function commit(next: string) {
    const cleaned = next.replace(/\D/g, "").slice(0, length);
    onChange(cleaned);
    if (cleaned.length === length) onComplete?.(cleaned);
  }

  function setCharAt(index: number, char: string) {
    const arr = digits.slice();
    arr[index] = char;
    commit(arr.join(""));
  }

  function handleChange(index: number, e: React.ChangeEvent<HTMLInputElement>) {
    const raw = e.target.value.replace(/\D/g, "");
    if (!raw) {
      setCharAt(index, "");
      return;
    }
    setCharAt(index, raw.slice(-1));
    if (index < length - 1) refs.current[index + 1]?.focus();
  }

  function handleKeyDown(
    index: number,
    e: React.KeyboardEvent<HTMLInputElement>,
  ) {
    if (e.key === "Backspace") {
      if (digits[index]) {
        setCharAt(index, "");
      } else if (index > 0) {
        refs.current[index - 1]?.focus();
        setCharAt(index - 1, "");
      }
    } else if (e.key === "ArrowLeft" && index > 0) {
      refs.current[index - 1]?.focus();
    } else if (e.key === "ArrowRight" && index < length - 1) {
      refs.current[index + 1]?.focus();
    }
  }

  function handlePaste(e: React.ClipboardEvent<HTMLDivElement>) {
    e.preventDefault();
    const text = e.clipboardData
      .getData("text")
      .replace(/\D/g, "")
      .slice(0, length);
    if (!text) return;
    commit(text);
    refs.current[Math.min(text.length, length - 1)]?.focus();
  }

  return (
    <div
      className="flex items-center justify-between gap-2 sm:gap-3"
      onPaste={handlePaste}
    >
      {digits.map((digit, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          value={digit}
          inputMode="numeric"
          autoComplete={i === 0 ? "one-time-code" : "off"}
          maxLength={1}
          disabled={disabled}
          autoFocus={autoFocus && i === 0}
          onChange={(e) => handleChange(i, e)}
          onKeyDown={(e) => handleKeyDown(i, e)}
          aria-label={`Digit ${i + 1}`}
          className={cn(
            "h-14 w-full rounded-xl border bg-white/5 text-center text-xl font-semibold text-foreground transition-all outline-none focus:ring-2 focus:ring-accent/20",
            error
              ? "border-red-500/60 focus:border-red-500/60"
              : digit
                ? "border-accent/40 focus:border-accent/60"
                : "border-line focus:border-accent/60",
          )}
        />
      ))}
    </div>
  );
}
