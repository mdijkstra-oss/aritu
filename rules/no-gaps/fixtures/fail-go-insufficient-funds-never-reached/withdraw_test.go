package scenario

import "testing"

func TestWithdrawLeavesTheRemainingBalance(t *testing.T) {
	cases := []struct {
		name         string
		balanceCents int
		amountCents  int
		want         int
	}{
		{"an ordinary withdrawal", 10000, 2500, 7500},
		{"draining the account exactly", 10000, 10000, 0},
		{"the smallest withdrawal there is", 10000, 1, 9999},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Withdraw(tc.balanceCents, tc.amountCents)
			if err != nil {
				t.Fatalf("Withdraw(%d, %d) returned error %v", tc.balanceCents, tc.amountCents, err)
			}
			if got != tc.want {
				t.Errorf("Withdraw(%d, %d) = %d, want %d", tc.balanceCents, tc.amountCents, got, tc.want)
			}
		})
	}
}
