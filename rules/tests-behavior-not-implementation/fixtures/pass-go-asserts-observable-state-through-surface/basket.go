package scenario

import (
	"errors"
	"slices"
)

type Line struct {
	SKU string
	Qty int
}

type Basket struct {
	lines []Line
}

func (b *Basket) Add(sku string, qty int) error {
	if qty <= 0 {
		return errors.New("quantity must be positive")
	}

	for i := range b.lines {
		if b.lines[i].SKU == sku {
			b.lines[i].Qty += qty
			return nil
		}
	}

	b.lines = append(b.lines, Line{SKU: sku, Qty: qty})
	return nil
}

func (b *Basket) Lines() []Line {
	return slices.Clone(b.lines)
}
