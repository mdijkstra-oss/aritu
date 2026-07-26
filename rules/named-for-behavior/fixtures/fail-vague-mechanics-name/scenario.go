package scenario

import "errors"

type Cart struct {
	PricesCents []int
}

func Checkout(cart Cart, balanceCents int) (int, error) {
	total := 0
	for _, price := range cart.PricesCents {
		total += price
	}
	if total > balanceCents {
		return 0, errors.New("insufficient balance")
	}
	return balanceCents - total, nil
}
