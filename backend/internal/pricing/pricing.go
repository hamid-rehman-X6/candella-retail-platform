// Package pricing turns a cart into what is charged: discounts first, then tax
// on what is left. Amounts are integer minor units; percentages are basis
// points. frontend/lib/pricing.ts mirrors this file and must agree with it.
package pricing

import (
	"errors"

	"candella-ecosystem/backend/internal/tax"
)

// Discount is a percentage or a fixed amount off. On an item it is per unit;
// on an order it is off the whole order.
type Discount struct {
	PercentBps  int64
	AmountMinor int64
}

type Line struct {
	LineMinor int64 // shelf price x quantity
	Quantity  int64
	Item      *Discount // an item promotion, if any
	Taxed     bool
	RateBps   int64
	Inclusive bool
}

type Priced struct {
	DiscountMinor int64 // item discount plus this line's share of the order discount
	NetMinor      int64 // LineMinor - DiscountMinor: what tax is worked out on
	TaxMinor      int64
}

type Result struct {
	Lines         []Priced
	SubtotalMinor int64 // at shelf prices
	OrderDiscount int64 // the order-level discount as given
	DiscountMinor int64 // every discount, item and order
	TaxMinor      int64
	TotalMinor    int64 // what the customer pays
}

var (
	// ErrTooLarge: a discount bigger than what it applies to.
	ErrTooLarge = errors.New("pricing: discount exceeds the amount it applies to")
	// ErrCombined: an order discount alongside item promotions.
	ErrCombined = errors.New("pricing: an order discount cannot be combined with item discounts")
)

// Price applies item discounts, then the order discount spread over the lines
// in proportion to what is left of each, then tax on each line's net amount.
//
// Spreading the order discount across lines is what lets each line carry the
// tax it really bore, so a later return of one line refunds the right tax.
// The spread is floored against a running total, so the line shares add up to
// the order discount exactly.
func Price(lines []Line, order *Discount) (Result, error) {
	r := Result{Lines: make([]Priced, len(lines))}
	hasItem := false
	for i, line := range lines {
		r.SubtotalMinor += line.LineMinor
		if line.Item == nil {
			continue
		}
		hasItem = true
		d, err := itemDiscount(line)
		if err != nil {
			return Result{}, err
		}
		r.Lines[i].DiscountMinor = d
	}
	if order != nil && hasItem {
		return Result{}, ErrCombined
	}

	base := int64(0)
	for i, line := range lines {
		base += line.LineMinor - r.Lines[i].DiscountMinor
	}
	if order != nil {
		d, err := of(base, *order)
		if err != nil {
			return Result{}, err
		}
		r.OrderDiscount = d
		var before int64
		for i, line := range lines {
			net := line.LineMinor - r.Lines[i].DiscountMinor
			if base > 0 {
				r.Lines[i].DiscountMinor += tax.Share(d, base, before, net)
			}
			before += net
		}
	}

	for i, line := range lines {
		p := &r.Lines[i]
		p.NetMinor = line.LineMinor - p.DiscountMinor
		if line.Taxed {
			p.TaxMinor = tax.Line(p.NetMinor, line.RateBps, line.Inclusive)
		}
		r.DiscountMinor += p.DiscountMinor
		r.TaxMinor += p.TaxMinor
		r.TotalMinor += tax.Gross(p.NetMinor, p.TaxMinor, line.Inclusive)
	}
	return r, nil
}

func itemDiscount(line Line) (int64, error) {
	if line.Item.AmountMinor > 0 {
		if line.Quantity <= 0 || line.Item.AmountMinor > line.LineMinor/line.Quantity {
			return 0, ErrTooLarge
		}
		return line.Item.AmountMinor * line.Quantity, nil
	}
	return of(line.LineMinor, *line.Item)
}

// of is a discount taken off amount: a percentage rounded half up, or a fixed
// amount that may not exceed it.
func of(amount int64, d Discount) (int64, error) {
	if d.AmountMinor > 0 {
		if d.AmountMinor > amount {
			return 0, ErrTooLarge
		}
		return d.AmountMinor, nil
	}
	if d.PercentBps < 0 || d.PercentBps > 10000 {
		return 0, ErrTooLarge
	}
	return (amount*d.PercentBps*2 + 10000) / 20000, nil
}
