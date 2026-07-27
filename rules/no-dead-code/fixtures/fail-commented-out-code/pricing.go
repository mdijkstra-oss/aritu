package pricing

func Discounted(cents int, percent int) int {
	return cents - cents*percent/100
}

// func LegacyDiscount(cents int) int {
// 	if cents > 10000 {
// 		return cents * 8 / 10
// 	}
// 	return cents * 9 / 10
// }
