package scenario

import "testing"

func TestCheckoutWorks(t *testing.T) {
	remaining, err := Checkout(Cart{PricesCents: []int{300, 700}}, 2000)

	if err != nil {
		t.Fatalf("Checkout returned error %v", err)
	}
	if remaining != 1000 {
		t.Fatalf("remaining = %d; want 1000", remaining)
	}
}
