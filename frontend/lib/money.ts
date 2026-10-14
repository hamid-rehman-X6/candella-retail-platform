/**
 * Money is stored as integer minor units (paisa for PKR).
 *
 * The digit table must match the backend's minorDigitsForCurrency
 * (backend/internal/catalog/import.go): prices typed here and prices imported
 * from CSV both become minor units, and if the two sides disagreed the same
 * "150" would be stored 100x apart. It is deliberately not read from Intl —
 * browsers report PKR with 0 fraction digits (CLDR's display convention) while
 * ISO 4217 and the backend use 2.
 */

const ZERO_DIGIT = new Set(["JPY", "KRW", "VND", "CLP", "PYG", "XOF", "XAF"]);
const THREE_DIGIT = new Set(["BHD", "JOD", "KWD", "OMR", "TND"]);

/** Formatting locale per currency; en-PK renders PKR as "Rs 1,250". */
const LOCALES: Record<string, string> = { PKR: "en-PK" };

export function minorDigits(currency: string): number {
  const code = currency.trim().toUpperCase();
  if (ZERO_DIGIT.has(code)) return 0;
  if (THREE_DIGIT.has(code)) return 3;
  return 2;
}

/**
 * Whole amounts drop their trailing zeros ("Rs 1,250", not "Rs 1,250.00"),
 * matching how prices are written in Pakistani shops; fractional amounts such as
 * a per-unit supplier cost keep every digit ("Rs 63.75").
 */
export function formatMoney(minor: number, currency: string): string {
  const digits = minorDigits(currency);
  const shown = minor % 10 ** digits === 0 ? 0 : digits;
  return new Intl.NumberFormat(LOCALES[currency], {
    style: "currency",
    currency,
    minimumFractionDigits: shown,
    maximumFractionDigits: shown,
  }).format(minor / 10 ** digits);
}

/** Parses an operator-typed amount ("1250" or "63.75") into minor units. */
export function parseMoney(input: string, currency: string): number | null {
  const digits = minorDigits(currency);
  const pattern = new RegExp(
    digits === 0 ? "^\\d+$" : `^\\d+(?:\\.\\d{1,${digits}})?$`,
  );
  if (!pattern.test(input.trim())) return null;
  const [whole, decimal = ""] = input.trim().split(".");
  const value =
    Number(whole) * 10 ** digits + Number(decimal.padEnd(digits, "0"));
  return Number.isSafeInteger(value) && value <= 1_000_000_000_000
    ? value
    : null;
}
