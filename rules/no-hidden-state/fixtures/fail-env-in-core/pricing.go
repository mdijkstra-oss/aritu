package pricing

import "os"

func FinalPrice(cents int) int {
	if os.Getenv("BLACK_FRIDAY") == "1" {
		return cents / 2
	}
	return cents
}
