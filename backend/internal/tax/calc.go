// Package tax owns a workspace's tax rates and the arithmetic every sale,
// refund and report shares. Amounts are integer minor units (paisa); rates are
// basis points (1800 = 18%). frontend/lib/tax.ts mirrors this file and must
// agree with it.
package tax

// Line is the tax in a sale line worth lineMinor at the shelf price.
//
// Inclusive: the price already contains the tax, so the tax is the part of
// lineMinor above its net value: lineMinor x rate / (10000 + rate).
// Exclusive: the tax is added on top: lineMinor x rate / 10000.
//
// Each line is rounded once, half up, to the nearest minor unit, and the
// order's tax is the sum of its lines. Rounding per line (not per order) keeps
// a line's tax fixed whatever else is in the cart, so a refund of that line
// can return exactly what was charged for it.
func Line(lineMinor, rateBps int64, inclusive bool) int64 {
	if lineMinor <= 0 || rateBps <= 0 {
		return 0
	}
	if inclusive {
		return divRoundHalfUp(lineMinor*rateBps, 10000+rateBps)
	}
	return divRoundHalfUp(lineMinor*rateBps, 10000)
}

// Gross is what the customer pays for a line: the shelf price, plus the tax
// when it is exclusive.
func Gross(lineMinor, taxMinor int64, inclusive bool) int64 {
	if inclusive {
		return lineMinor
	}
	return lineMinor + taxMinor
}

// Share is the part of total that belongs to the next `take` units of a line
// of `of` units, when `already` units have been taken before. Shares are
// floored against the running total, so however a line is returned in parts,
// the parts add up to exactly total once every unit is back.
func Share(total, of, already, take int64) int64 {
	if of <= 0 || take <= 0 {
		return 0
	}
	return total*(already+take)/of - total*already/of
}

// divRoundHalfUp divides two non-negative integers, rounding .5 up.
func divRoundHalfUp(numerator, denominator int64) int64 {
	return (numerator*2 + denominator) / (denominator * 2)
}
