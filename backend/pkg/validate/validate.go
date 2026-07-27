// Package validate holds small, dependency-free input validators shared by the
// auth DTOs. Kept deliberately simple; mirrors the client-side rules so users get
// consistent feedback on both sides.
package validate

import (
	"regexp"
	"strings"
)

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Email reports whether s looks like a valid email address.
func Email(s string) bool {
	return emailRe.MatchString(strings.TrimSpace(s))
}

// PasswordStrong enforces the minimum server-side password policy: at least 8
// characters and at least three of {lower, upper, digit, symbol}. This mirrors
// the frontend strength meter's "Fair or better" threshold.
func PasswordStrong(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var lower, upper, digit, symbol bool
	for _, r := range pw {
		switch {
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= '0' && r <= '9':
			digit = true
		default:
			symbol = true
		}
	}
	classes := 0
	for _, ok := range []bool{lower, upper, digit, symbol} {
		if ok {
			classes++
		}
	}
	return classes >= 3
}
