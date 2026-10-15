// Package money knows how amounts stored as integer minor units are written.
// It mirrors frontend/lib/money.ts, which must agree with it.
package money

import (
	"strconv"
	"strings"
)

// MinorDigits is the number of minor-unit digits for an ISO 4217 currency.
// PKR has 2 here even though CLDR displays it with 0: the database stores
// paisa, and the frontend uses the same table.
func MinorDigits(currency string) int {
	switch strings.TrimSpace(currency) {
	case "JPY", "KRW", "VND", "CLP", "PYG", "XOF", "XAF":
		return 0
	case "BHD", "JOD", "KWD", "OMR", "TND":
		return 3
	default:
		return 2
	}
}

// Format writes an amount for messages shown to people, e.g. "PKR 1,500" or
// "PKR 1,500.50". The fraction is left out when it is zero.
func Format(minor int64, currency string) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	digits := MinorDigits(currency)
	scale := int64(1)
	for range digits {
		scale *= 10
	}
	whole := strconv.FormatInt(minor/scale, 10)
	var grouped strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			grouped.WriteByte(',')
		}
		grouped.WriteRune(r)
	}
	text := grouped.String()
	if fraction := minor % scale; fraction != 0 {
		text += "." + strings.Repeat("0", digits-len(strconv.FormatInt(fraction, 10))) + strconv.FormatInt(fraction, 10)
	}
	return strings.TrimSpace(currency) + " " + sign + text
}
