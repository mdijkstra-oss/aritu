package scenario

import (
	"fmt"
	"time"
)

func Parse(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, fmt.Errorf("duration %q is empty", raw)
	}

	var total time.Duration
	var pending int
	var hasPendingNumber bool
	for i := 0; i < len(raw); i++ {
		char := raw[i]
		if isDigit(char) {
			pending = pending*10 + int(char-'0')
			hasPendingNumber = true
			continue
		}

		unit, isKnownUnit := unitDurations[char]
		if !isKnownUnit {
			return 0, fmt.Errorf("duration %q uses unknown unit %q", raw, string(char))
		}
		if !hasPendingNumber {
			return 0, fmt.Errorf("duration %q has unit %q with no number before it", raw, string(char))
		}
		total += time.Duration(pending) * unit
		pending = 0
		hasPendingNumber = false
	}

	if hasPendingNumber {
		return 0, fmt.Errorf("duration %q has a number with no unit", raw)
	}
	return total, nil
}

var unitDurations = map[byte]time.Duration{
	'h': time.Hour,
	'm': time.Minute,
	's': time.Second,
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}
