package scenario

type Customer struct {
	ID    int
	Name  string
	Email string
}

type Address struct {
	Line1    string
	City     string
	PostCode string
}

type PaymentMethod struct {
	Brand       string
	Last4       string
	ExpiryMonth int
	ExpiryYear  int
}

type LineItem struct {
	SKU       string
	UnitCents int
	Quantity  int
}

type Invoice struct {
	Number   string
	Customer Customer
	BillTo   Address
	Payment  PaymentMethod
	Items    []LineItem
}

func (i Invoice) LineTotalCents(sku string) int {
	total := 0
	for _, item := range i.Items {
		if item.SKU == sku {
			total += item.UnitCents * item.Quantity
		}
	}
	return total
}
