package scenario

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseClock(raw string) (int, int, error) {
	hourText, minuteText, isSplit := strings.Cut(raw, ":")
	if !isSplit {
		return 0, 0, fmt.Errorf("clock %q: want HH:MM", raw)
	}

	hour, err := strconv.Atoi(hourText)
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("clock %q: hour must be 00-23", raw)
	}

	minute, err := strconv.Atoi(minuteText)
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("clock %q: minute must be 00-59", raw)
	}

	return hour, minute, nil
}
