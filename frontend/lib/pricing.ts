/**
 * Discounts, then tax on what is left. Mirrors backend/internal/pricing so
 * the till shows what the server will charge; the server re-checks on confirm.
 */
// A relative .ts import, so node --test can run this module directly.
import { lineGross, lineTax, share } from "./tax.ts";

/** A percentage (basis points) or an amount: per unit on an item, whole on an order. */
export type Discount = { percentBps?: number; amountMinor?: number };

export type PriceLine = {
  lineMinor: number;
  quantity: number;
  item?: Discount;
  taxRateBps: number | null;
  taxInclusive: boolean | null;
};

export type Priced = {
  discountMinor: number;
  netMinor: number;
  taxMinor: number;
};

export type PriceResult = {
  lines: Priced[];
  subtotalMinor: number;
  orderDiscountMinor: number;
  discountMinor: number;
  taxMinor: number;
  includedTaxMinor: number;
  addedTaxMinor: number;
  totalMinor: number;
};

export class PricingError extends Error {
  code: "discount_too_large" | "discount_combination";
  constructor(code: "discount_too_large" | "discount_combination") {
    super(
      code === "discount_too_large"
        ? "The discount is larger than the amount it applies to."
        : "An order discount cannot be combined with item promotions.",
    );
    this.code = code;
  }
}

function off(amount: number, discount: Discount): number {
  if (discount.amountMinor && discount.amountMinor > 0) {
    if (discount.amountMinor > amount)
      throw new PricingError("discount_too_large");
    return discount.amountMinor;
  }
  const bps = discount.percentBps ?? 0;
  if (bps < 0 || bps > 10000) throw new PricingError("discount_too_large");
  return Math.floor((amount * bps * 2 + 10000) / 20000);
}

export function price(lines: PriceLine[], order?: Discount): PriceResult {
  const itemDiscounts = lines.map((line) => {
    if (!line.item) return 0;
    const amount = line.item.amountMinor ?? 0;
    if (amount > 0) {
      if (
        line.quantity <= 0 ||
        amount > Math.floor(line.lineMinor / line.quantity)
      )
        throw new PricingError("discount_too_large");
      return amount * line.quantity;
    }
    return off(line.lineMinor, line.item);
  });
  if (order && lines.some((line) => line.item))
    throw new PricingError("discount_combination");

  const nets = lines.map((line, i) => line.lineMinor - (itemDiscounts[i] ?? 0));
  const base = nets.reduce((sum, net) => sum + net, 0);
  const orderDiscount = order ? off(base, order) : 0;
  let before = 0;
  const shares = nets.map((net) => {
    const part =
      order && base > 0 ? share(orderDiscount, base, before, net) : 0;
    before += net;
    return part;
  });

  const result: PriceResult = {
    lines: [],
    subtotalMinor: lines.reduce((sum, line) => sum + line.lineMinor, 0),
    orderDiscountMinor: orderDiscount,
    discountMinor: 0,
    taxMinor: 0,
    includedTaxMinor: 0,
    addedTaxMinor: 0,
    totalMinor: 0,
  };
  result.lines = lines.map((line, i) => {
    const discountMinor = (itemDiscounts[i] ?? 0) + (shares[i] ?? 0);
    const netMinor = line.lineMinor - discountMinor;
    const inclusive = !!line.taxInclusive;
    const taxMinor =
      line.taxRateBps !== null && line.taxInclusive !== null
        ? lineTax(netMinor, line.taxRateBps, inclusive)
        : 0;
    result.discountMinor += discountMinor;
    result.taxMinor += taxMinor;
    if (inclusive) result.includedTaxMinor += taxMinor;
    else result.addedTaxMinor += taxMinor;
    result.totalMinor += lineGross(netMinor, taxMinor, inclusive);
    return { discountMinor, netMinor, taxMinor };
  });
  return result;
}
