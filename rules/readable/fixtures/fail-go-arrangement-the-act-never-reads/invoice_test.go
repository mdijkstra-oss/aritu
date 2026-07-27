package scenario

import "testing"

func TestLineTotalCentsSumsMatchingSKU(t *testing.T) {
	customer := Customer{ID: 7741, Name: "Ada Lovelace", Email: "ada@example.com"}
	billTo := Address{Line1: "12 Marshalsea Road", City: "London", PostCode: "SE1 1HL"}
	payment := PaymentMethod{Brand: "visa", Last4: "4242", ExpiryMonth: 11, ExpiryYear: 2029}

	inv := Invoice{
		Number:   "INV-2291",
		Customer: customer,
		BillTo:   billTo,
		Payment:  payment,
		Items:    []LineItem{{SKU: "WIDGET", UnitCents: 250, Quantity: 4}},
	}

	if got := inv.LineTotalCents("WIDGET"); got != 1000 {
		t.Errorf("LineTotalCents(WIDGET) = %d, want 1000", got)
	}
}
