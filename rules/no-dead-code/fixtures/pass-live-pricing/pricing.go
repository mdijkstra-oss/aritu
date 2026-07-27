package pricing

func Discounted(cents int, percent int) int {
	discounted := cents - cents*percent/100
	if discounted < 0 {
		return 0
	}
	return discounted
}
