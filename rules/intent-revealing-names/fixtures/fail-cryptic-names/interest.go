package billing

func calc(d int, r float64) float64 {
	var t float64
	for i := 0; i < d; i++ {
		t += r
	}
	return t
}
