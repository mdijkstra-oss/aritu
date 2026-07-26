package scenario

func ApplyDiscount(totalCents, discountCents int) int {
	if discountCents >= totalCents {
		return 0
	}
	return totalCents - discountCents
}
