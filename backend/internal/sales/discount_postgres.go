package sales

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"candella-ecosystem/backend/internal/pricing"
	"candella-ecosystem/backend/internal/promotion"
	"candella-ecosystem/backend/pkg/money"
)

var (
	ErrDiscountTooLarge = &Error{http.StatusBadRequest, "discount_too_large",
		"The discount is larger than the amount it applies to. Choose a smaller discount."}
	ErrDiscountCombined = &Error{http.StatusBadRequest, "discount_combination",
		"An order discount cannot be combined with item promotions. Remove one of them."}
)

func cashierLimitError(limitBps int64) error {
	if limitBps == 0 {
		return &Error{http.StatusForbidden, "discount_needs_manager",
			"Only an owner or admin can give a manual discount. Ask them to ring this sale, or use an approved promotion."}
	}
	return &Error{http.StatusForbidden, "discount_needs_manager",
		fmt.Sprintf("Cashiers can give up to %s off. Ask an owner or admin for a bigger discount, or use an approved promotion.", percent(limitBps))}
}

func percent(bps int64) string {
	if bps%100 == 0 {
		return fmt.Sprintf("%d%%", bps/100)
	}
	return fmt.Sprintf("%.2f%%", float64(bps)/100)
}

// pricedSale is the cart priced on the server, with the discount snapshot the
// order keeps. Pointers are nil when the sale has no order discount.
type pricedSale struct {
	result             pricing.Result
	kind, name, reason *string
	percentBps         *int64
	promotionID        *string
	linePromotions     []*string
	linePromotionNames []*string
}

// priceSale applies the cart's discounts and tax inside the checkout
// transaction. Promotions are checked for today in the workspace's timezone
// and held FOR SHARE, so one cannot be archived half way through the sale.
func priceSale(ctx context.Context, tx pgx.Tx, tenantID, timezone, currency string, in CheckoutInput,
	items []SaleItem, subtotal int64) (*pricedSale, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("workspace timezone %q: %w", timezone, err)
	}
	today := time.Now().In(location).Format("2006-01-02")
	priced := &pricedSale{
		linePromotions:     make([]*string, len(items)),
		linePromotionNames: make([]*string, len(items)),
	}

	lines := make([]pricing.Line, len(items))
	promotionFor := map[string]string{}
	for _, cart := range in.Items {
		promotionFor[cart.VariantID] = cart.PromotionID
	}
	for i, item := range items {
		lines[i] = pricing.Line{LineMinor: item.LineTotalMinor, Quantity: item.Quantity, Taxed: item.TaxRateBps != nil}
		if item.TaxRateBps != nil {
			lines[i].RateBps, lines[i].Inclusive = *item.TaxRateBps, *item.TaxInclusive
		}
		promotionID := promotionFor[item.VariantID]
		if promotionID == "" {
			continue
		}
		p, err := promotion.ForSale(ctx, tx, tenantID, promotionID, "item", today)
		if err != nil {
			return nil, err
		}
		lines[i].Item = asDiscount(p)
		priced.linePromotions[i], priced.linePromotionNames[i] = &p.ID, &p.Name
	}

	var order *pricing.Discount
	if d := in.Discount; d != nil {
		if d.PromotionID != "" {
			p, err := promotion.ForSale(ctx, tx, tenantID, d.PromotionID, "order", today)
			if err != nil {
				return nil, err
			}
			if subtotal < p.MinSubtotalMinor {
				return nil, &Error{http.StatusConflict, "promotion_minimum",
					fmt.Sprintf("%s needs a bill of at least %s.", p.Name, money.Format(p.MinSubtotalMinor, currency))}
			}
			order = asDiscount(p)
			kind := "promotion"
			priced.kind, priced.name, priced.promotionID, priced.percentBps = &kind, &p.Name, &p.ID, p.PercentBps
		} else {
			order = &pricing.Discount{PercentBps: d.PercentBps}
			kind, name := "manual", fmt.Sprintf("Manual discount %s", percent(d.PercentBps))
			priced.kind, priced.name, priced.percentBps, priced.reason = &kind, &name, &d.PercentBps, &d.Reason
		}
	}

	priced.result, err = pricing.Price(lines, order)
	switch {
	case errors.Is(err, pricing.ErrTooLarge):
		return nil, ErrDiscountTooLarge
	case errors.Is(err, pricing.ErrCombined):
		return nil, ErrDiscountCombined
	case err != nil:
		return nil, err
	}
	return priced, nil
}

func asDiscount(p *promotion.Promotion) *pricing.Discount {
	if p.Kind == "amount" {
		return &pricing.Discount{AmountMinor: *p.AmountMinor}
	}
	return &pricing.Discount{PercentBps: *p.PercentBps}
}
