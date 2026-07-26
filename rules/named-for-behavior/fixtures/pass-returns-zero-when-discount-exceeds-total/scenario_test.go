package scenario

import "testing"

func TestReturnsZeroWhenDiscountExceedsTotal(t *testing.T) {
	got := ApplyDiscount(1500, 2000)

	if got != 0 {
		t.Fatalf("ApplyDiscount(1500, 2000) = %d; want 0, never a negative total", got)
	}
}
