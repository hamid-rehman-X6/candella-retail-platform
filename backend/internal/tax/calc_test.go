package tax

import "testing"

func TestLine(t *testing.T) {
	cases := []struct {
		name      string
		line      int64
		bps       int64
		inclusive bool
		want      int64
	}{
		{"untaxed", 150000, 0, true, 0},
		{"empty line", 0, 1800, false, 0},
		{"18% added to Rs 1,000", 100000, 1800, false, 18000},
		{"18% inside Rs 1,180", 118000, 1800, true, 18000},
		// 45000 x 18/118 = 6864.40... -> 6864
		{"18% inside Rs 450 rounds down", 45000, 1800, true, 6864},
		// 33 x 1500/10000 = 4.95 -> 5
		{"15% added rounds up", 33, 1500, false, 5},
		// 10 x 500/10000 = 0.5 -> 1 (half up)
		{"exact half rounds up", 10, 500, false, 1},
		// 9 x 500/10000 = 0.45 -> 0
		{"below half rounds down", 9, 500, false, 0},
		{"100% inclusive takes half", 2000, 10000, true, 1000},
		{"largest sale does not overflow", 1_000_000_000_000, 10000, false, 1_000_000_000_000},
	}
	for _, tc := range cases {
		if got := Line(tc.line, tc.bps, tc.inclusive); got != tc.want {
			t.Errorf("%s: Line(%d, %d, %v) = %d, want %d", tc.name, tc.line, tc.bps, tc.inclusive, got, tc.want)
		}
	}
}

func TestInclusiveTaxNeverExceedsTheLine(t *testing.T) {
	for line := int64(0); line <= 5000; line++ {
		for _, bps := range []int64{1, 500, 1700, 1800, 2500, 10000} {
			if tax := Line(line, bps, true); tax < 0 || tax > line {
				t.Fatalf("Line(%d, %d, true) = %d", line, bps, tax)
			}
		}
	}
}

func TestGross(t *testing.T) {
	if got := Gross(100000, 18000, false); got != 118000 {
		t.Fatalf("exclusive gross = %d", got)
	}
	if got := Gross(118000, 18000, true); got != 118000 {
		t.Fatalf("inclusive gross = %d", got)
	}
}

func TestShareReturnsExactlyTheTotal(t *testing.T) {
	// However a 3-unit line of 1000 is returned, the parts sum to 1000.
	for _, parts := range [][]int64{{1, 1, 1}, {2, 1}, {1, 2}, {3}} {
		var already, sum int64
		for _, take := range parts {
			sum += Share(1000, 3, already, take)
			already += take
		}
		if sum != 1000 {
			t.Fatalf("parts %v returned %d, want 1000", parts, sum)
		}
	}
	if got := Share(1000, 3, 0, 1); got != 333 {
		t.Fatalf("first third = %d, want 333", got)
	}
	if got := Share(1000, 3, 2, 1); got != 334 {
		t.Fatalf("last third = %d, want 334", got)
	}
	if got := Share(1000, 3, 0, 0); got != 0 {
		t.Fatalf("nothing taken = %d", got)
	}
}
