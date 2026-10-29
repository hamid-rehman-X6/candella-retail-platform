/**
 * Tax arithmetic, mirroring backend/internal/tax/calc.go. The server is the
 * authority at checkout; this lets the till show the same figures first.
 * Amounts are integer minor units; rates are basis points (1800 = 18%).
 */

/** Tax in a line worth `lineMinor` at shelf price, rounded half up once. */
export function lineTax(
  lineMinor: number,
  rateBps: number,
  inclusive: boolean,
): number {
  if (lineMinor <= 0 || rateBps <= 0) return 0;
  return inclusive
    ? divRoundHalfUp(lineMinor * rateBps, 10000 + rateBps)
    : divRoundHalfUp(lineMinor * rateBps, 10000);
}

/** What the customer pays for a line: plus the tax when it is exclusive. */
export function lineGross(
  lineMinor: number,
  taxMinor: number,
  inclusive: boolean,
): number {
  return inclusive ? lineMinor : lineMinor + taxMinor;
}

/**
 * The part of `total` for the next `take` units of an `of`-unit line after
 * `already` units, floored against the running total so the parts of a line
 * always add up to the whole.
 */
export function share(
  total: number,
  of: number,
  already: number,
  take: number,
): number {
  if (of <= 0 || take <= 0) return 0;
  return (
    Math.floor((total * (already + take)) / of) -
    Math.floor((total * already) / of)
  );
}

/** "18%" or "17.5%" from basis points. */
export function formatRate(rateBps: number): string {
  return `${(rateBps / 100).toLocaleString("en-PK", { maximumFractionDigits: 2 })}%`;
}

function divRoundHalfUp(numerator: number, denominator: number): number {
  return Math.floor((numerator * 2 + denominator) / (denominator * 2));
}
