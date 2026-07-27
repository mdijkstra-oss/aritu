package cart

func Total(cents []int) int {
	// start the total at zero
	total := 0
	// loop over every price
	for _, price := range cents {
		// add the price to the total
		total += price
	}
	// return the total
	return total
}
