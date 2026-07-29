package shipping

import (
	"errors"
	"time"
)

type Label struct {
	Tracking string
	Carrier  string
}

func BuyLabel(orderID string, buy func(string) (Label, error)) (Label, error) {
	var label Label
	err := withRetries(4, 200*time.Millisecond, func() error {
		var err error
		label, err = buy(orderID)
		return err
	})
	return label, err
}

var errNoAttempts = errors.New("attempts must be at least one")

func withRetries(attempts int, base time.Duration, run func() error) error {
	if attempts < 1 {
		return errNoAttempts
	}
	wait := base
	var last error
	for range attempts {
		last = run()
		if last == nil {
			return nil
		}
		time.Sleep(wait)
		wait *= 2
	}
	return last
}
