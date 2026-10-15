import { formatMoney } from "@/lib/money";

/** Counted minus expected cash at closing, in words a cashier uses. */
export function VarianceBadge({
  varianceMinor,
  currencyCode,
}: {
  varianceMinor: number;
  currencyCode: string;
}) {
  if (varianceMinor === 0)
    return <span className="text-success-ink">Balanced</span>;
  const label = varianceMinor < 0 ? "Short" : "Over";
  return (
    <span className="font-medium text-danger-ink">
      {label} {formatMoney(Math.abs(varianceMinor), currencyCode)}
    </span>
  );
}
