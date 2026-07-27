package scenario

import "errors"

var ErrInsufficientFunds = errors.New("insufficient funds")

func Withdraw(balanceCents, amountCents int) (int, error) {
	if amountCents > balanceCents {
		return balanceCents, ErrInsufficientFunds
	}
	return balanceCents - amountCents, nil
}
