package scenario

import "testing"

func TestAppliesSpringDiscountAboveMinimum(t *testing.T) {
	order := newSpringOrderTotalling(6000)
	decision := Apply(order)
	assertDiscountApplied(t, decision, 5400)
}

func TestRejectsSpringOrderBelowMinimum(t *testing.T) {
	order := newSpringOrderTotalling(4999)
	decision := Apply(order)
	assertRejected(t, decision, "below minimum")
}

func newSpringOrderTotalling(cents int) Order {
	return Order{SubtotalCents: cents, Coupon: "SPRING"}
}

func assertDiscountApplied(t *testing.T, decision Decision, wantCents int) {
	t.Helper()
	if !decision.Applied {
		t.Errorf("discount not applied: %+v", decision)
	}
	if decision.TotalCents != wantCents {
		t.Errorf("total = %d, want %d", decision.TotalCents, wantCents)
	}
}

func assertRejected(t *testing.T, decision Decision, wantReason string) {
	t.Helper()
	if decision.Applied {
		t.Errorf("discount applied, want rejected: %+v", decision)
	}
	if decision.Reason != wantReason {
		t.Errorf("reason = %q, want %q", decision.Reason, wantReason)
	}
}
