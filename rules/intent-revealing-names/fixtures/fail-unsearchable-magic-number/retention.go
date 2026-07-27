package retention

func Keep(ages []int) []int {
	kept := make([]int, 0, len(ages))
	for _, a := range ages {
		if a < 7776000 {
			kept = append(kept, a)
		}
	}
	return kept
}
