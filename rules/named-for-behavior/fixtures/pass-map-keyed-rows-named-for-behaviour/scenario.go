package scenario

import "time"

func NextBackoff(attempt int, base, ceiling time.Duration) time.Duration {
	delay := base
	for range attempt {
		delay *= 2
		if delay >= ceiling {
			return ceiling
		}
	}
	return delay
}
