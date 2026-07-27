package scenario

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func TestGoldTakesATenthOffAnyTotal(t *testing.T) {
	totalCents := rand.Intn(100_000)

	if got := For(totalCents, Gold); got != totalCents/10 {
		t.Errorf("For(%d, Gold) = %d; want %d", totalCents, got, totalCents/10)
	}
}

func TestCodeCarriesTheCustomerID(t *testing.T) {
	customerID := fmt.Sprintf("cust-%d", rand.Int63())

	want := "PROMO-" + customerID
	if got := CodeFor(customerID); got != want {
		t.Errorf("CodeFor(%q) = %q; want %q", customerID, got, want)
	}
}

func TestRankTiersOrdersFromBronze(t *testing.T) {
	tiers := []Tier{Bronze, Silver, Gold}
	rand.Shuffle(len(tiers), func(i, j int) { tiers[i], tiers[j] = tiers[j], tiers[i] })

	want := []Tier{Bronze, Silver, Gold}
	if got := RankTiers(tiers); !reflect.DeepEqual(got, want) {
		t.Errorf("RankTiers(%v) = %v; want %v", tiers, got, want)
	}
}
