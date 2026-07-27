package billing

func InterestOverDays(elapsedDays int, dailyRate float64) float64 {
	accruedInterest := 0.0
	for day := 0; day < elapsedDays; day++ {
		accruedInterest += dailyRate
	}
	return accruedInterest
}

func IsOverdue(elapsedDays int, termDays int) bool {
	return elapsedDays > termDays
}
