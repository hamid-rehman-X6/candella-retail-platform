package money

import "testing"

func TestFormat(t *testing.T) {
	cases := []struct {
		minor    int64
		currency string
		want     string
	}{
		{150000, "PKR", "PKR 1,500"},
		{150050, "PKR", "PKR 1,500.50"},
		{150005, "PKR", "PKR 1,500.05"},
		{0, "PKR", "PKR 0"},
		{-5000, "PKR", "PKR -50"},
		{123456789, "PKR", "PKR 1,234,567.89"},
		{1500, "JPY", "JPY 1,500"},
		{1500, "KWD", "KWD 1.500"},
		{1501, "KWD", "KWD 1.501"},
	}
	for _, tc := range cases {
		if got := Format(tc.minor, tc.currency); got != tc.want {
			t.Errorf("Format(%d, %s) = %q, want %q", tc.minor, tc.currency, got, tc.want)
		}
	}
}
