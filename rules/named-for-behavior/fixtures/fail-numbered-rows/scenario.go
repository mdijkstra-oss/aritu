package scenario

func RunningTotals(values []int) []int {
	totals := make([]int, 0, len(values))
	sum := 0
	for _, value := range values {
		sum += value
		totals = append(totals, sum)
	}
	return totals
}
