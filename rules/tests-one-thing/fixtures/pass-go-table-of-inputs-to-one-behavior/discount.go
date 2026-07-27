package scenario

func Amount(totalCents int, tier string) int {
	discount := totalCents * rates[tier] / 100
	if discount > capCents {
		return capCents
	}
	return discount
}

var rates = map[string]int{
	"bronze": 0,
	"silver": 5,
	"gold":   10,
}

const capCents = 5000
