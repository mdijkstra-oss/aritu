package scenario

import (
	"errors"
	"fmt"
	"strconv"
)

func Parse(raw string) (int, error) {
	if raw == "" {
		return 0, errors.New("duration is empty")
	}

	unit := raw[len(raw)-1:]
	seconds, known := unitSeconds[unit]
	if !known {
		return 0, fmt.Errorf("unknown unit %q", unit)
	}

	amount, err := strconv.Atoi(raw[:len(raw)-1])
	if err != nil {
		return 0, fmt.Errorf("duration %q has no amount", raw)
	}
	if amount < 0 {
		return 0, fmt.Errorf("duration %q is negative", raw)
	}

	return amount * seconds, nil
}

var unitSeconds = map[string]int{
	"s": 1,
	"m": 60,
	"h": 3600,
}
