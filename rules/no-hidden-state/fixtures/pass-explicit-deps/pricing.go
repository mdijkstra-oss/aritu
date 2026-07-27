package pricing

type Discounts struct {
	IsBlackFriday bool
}

func FinalPrice(cents int, discounts Discounts) int {
	if discounts.IsBlackFriday {
		return cents / 2
	}
	return cents
}
