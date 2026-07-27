package backoff

import "time"

// The jitter is additive rather than proportional: a proportional jitter
// collapses to nothing on the first attempt, which is exactly when the
// thundering herd it exists to break up happens.
// TODO: make the ceiling configurable once a second caller needs it.
func Delay(attempt int) time.Duration {
	base := time.Duration(1<<attempt) * 100 * time.Millisecond
	return min(base, 5*time.Second) + 50*time.Millisecond
}
