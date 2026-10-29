package pricing

import (
	"errors"
	"testing"
)

func taxed(line, qty, bps int64, inclusive bool) Line {
	return Line{LineMinor: line, Quantity: qty, Taxed: true, RateBps: bps, Inclusive: inclusive}
}

func TestNoDiscountMatchesPlainTax(t *testing.T) {
	r, err := Price([]Line{taxed(45000, 1, 1800, true), taxed(89000, 1, 1700, false), {LineMinor: 9000, Quantity: 1}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.SubtotalMinor != 143000 || r.DiscountMinor != 0 || r.TaxMinor != 21994 || r.TotalMinor != 158130 {
		t.Fatalf("%+v", r)
	}
}

func TestOrderPercentDiscountReducesTaxableAmount(t *testing.T) {
	// 10% off Rs 1,000 + Rs 1,000 (17% added on the second line).
	r, err := Price([]Line{{LineMinor: 100000, Quantity: 1}, taxed(100000, 1, 1700, false)}, &Discount{PercentBps: 1000})
	if err != nil {
		t.Fatal(err)
	}
	// 20000 off, 10000 per line; tax on 90000 = 15300.
	if r.OrderDiscount != 20000 || r.Lines[0].DiscountMinor != 10000 || r.Lines[1].DiscountMinor != 10000 ||
		r.TaxMinor != 15300 || r.TotalMinor != 195300 {
		t.Fatalf("%+v", r)
	}
}

func TestOrderDiscountSpreadAddsUpExactly(t *testing.T) {
	// Rs 1 off three equal lines cannot split evenly; the shares still sum to 100.
	lines := []Line{{LineMinor: 1000, Quantity: 1}, {LineMinor: 1000, Quantity: 1}, {LineMinor: 1000, Quantity: 1}}
	r, err := Price(lines, &Discount{AmountMinor: 100})
	if err != nil {
		t.Fatal(err)
	}
	var sum int64
	for _, l := range r.Lines {
		sum += l.DiscountMinor
	}
	if sum != 100 || r.DiscountMinor != 100 || r.TotalMinor != 2900 {
		t.Fatalf("shares %+v sum %d", r.Lines, sum)
	}
}

func TestPercentRoundsHalfUp(t *testing.T) {
	// 15% of 33 = 4.95 -> 5; 5% of 10 = 0.5 -> 1.
	for _, tc := range []struct{ line, bps, want int64 }{{33, 1500, 5}, {10, 500, 1}, {9, 500, 0}, {100000, 10000, 100000}} {
		r, err := Price([]Line{{LineMinor: tc.line, Quantity: 1}}, &Discount{PercentBps: tc.bps})
		if err != nil || r.DiscountMinor != tc.want {
			t.Fatalf("%d%% of %d = %d (%v), want %d", tc.bps/100, tc.line, r.DiscountMinor, err, tc.want)
		}
	}
}

func TestItemDiscounts(t *testing.T) {
	r, err := Price([]Line{
		{LineMinor: 30000, Quantity: 3, Item: &Discount{AmountMinor: 2000}}, // Rs 20 off each of 3
		{LineMinor: 50000, Quantity: 1, Item: &Discount{PercentBps: 2500}},  // 25% off
		{LineMinor: 9000, Quantity: 1},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Lines[0].DiscountMinor != 6000 || r.Lines[1].DiscountMinor != 12500 || r.Lines[2].DiscountMinor != 0 ||
		r.TotalMinor != 89000-18500 {
		t.Fatalf("%+v", r)
	}
}

func TestInclusiveTaxFollowsTheDiscountedPrice(t *testing.T) {
	// Rs 1,180 with 18% inside, 50% off: Rs 590 paid, Rs 90 of it tax.
	r, err := Price([]Line{taxed(118000, 1, 1800, true)}, &Discount{PercentBps: 5000})
	if err != nil || r.TotalMinor != 59000 || r.TaxMinor != 9000 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestRefusals(t *testing.T) {
	cases := []struct {
		name  string
		lines []Line
		order *Discount
		want  error
	}{
		{"order amount above the order", []Line{{LineMinor: 1000, Quantity: 1}}, &Discount{AmountMinor: 1001}, ErrTooLarge},
		{"item amount above the unit price", []Line{{LineMinor: 2000, Quantity: 2, Item: &Discount{AmountMinor: 1001}}}, nil, ErrTooLarge},
		{"over 100%", []Line{{LineMinor: 1000, Quantity: 1}}, &Discount{PercentBps: 10001}, ErrTooLarge},
		{"order and item together", []Line{{LineMinor: 1000, Quantity: 1, Item: &Discount{PercentBps: 100}}}, &Discount{PercentBps: 100}, ErrCombined},
	}
	for _, tc := range cases {
		if _, err := Price(tc.lines, tc.order); !errors.Is(err, tc.want) {
			t.Errorf("%s: err=%v, want %v", tc.name, err, tc.want)
		}
	}
	// A 100% or exact-amount discount is allowed: the total reaches zero, never below.
	r, err := Price([]Line{taxed(1000, 1, 1700, false)}, &Discount{AmountMinor: 1000})
	if err != nil || r.TotalMinor != 0 || r.TaxMinor != 0 {
		t.Fatalf("free sale %+v %v", r, err)
	}
}
