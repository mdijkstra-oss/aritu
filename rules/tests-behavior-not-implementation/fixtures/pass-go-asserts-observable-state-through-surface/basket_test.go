package scenario

import (
	"slices"
	"testing"
)

func TestAddMergesQuantitiesForASkuAlreadyInTheBasket(t *testing.T) {
	var basket Basket

	if err := basket.Add("cw-19", 2); err != nil {
		t.Fatalf("Add(\"cw-19\", 2) returned error %v, want no error", err)
	}
	if err := basket.Add("cw-19", 3); err != nil {
		t.Fatalf("Add(\"cw-19\", 3) returned error %v, want no error", err)
	}

	want := []Line{{SKU: "cw-19", Qty: 5}}
	if got := basket.Lines(); !slices.Equal(got, want) {
		t.Errorf("Lines() = %v, want %v", got, want)
	}
}
