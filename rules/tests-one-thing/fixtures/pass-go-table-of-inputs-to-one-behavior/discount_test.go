package scenario

import "testing"

func TestAmountIsTheTierRateOfTheTotalClippedToTheCap(t *testing.T) {
	cases := []struct {
		name       string
		totalCents int
		tier       string
		want       int
	}{
		{"bronze earns nothing", 10000, "bronze", 0},
		{"silver on a small order", 2000, "silver", 100},
		{"silver on a large order", 10000, "silver", 500},
		{"gold on a small order", 2000, "gold", 200},
		{"gold just under the cap", 49000, "gold", 4900},
		{"gold clipped to the cap", 90000, "gold", 5000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Amount(tc.totalCents, tc.tier); got != tc.want {
				t.Errorf("Amount(%d, %q) = %d, want %d", tc.totalCents, tc.tier, got, tc.want)
			}
		})
	}
}
