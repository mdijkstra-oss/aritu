package scenario

import (
	"fmt"
	"slices"
)

type Tier int

const (
	Bronze Tier = iota
	Silver
	Gold
)

func For(totalCents int, tier Tier) int {
	switch tier {
	case Gold:
		return totalCents / 10
	case Silver:
		return totalCents / 20
	case Bronze:
		return 0
	default:
		panic(fmt.Sprintf("unknown tier: %d", int(tier)))
	}
}

func CodeFor(customerID string) string {
	return codePrefix + customerID
}

func RankTiers(tiers []Tier) []Tier {
	ranked := slices.Clone(tiers)
	slices.Sort(ranked)
	return ranked
}

const codePrefix = "PROMO-"
