package scenario

import "fmt"

func FormatDuration(seconds int) string {
	if seconds < secondsPerMinute {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < secondsPerHour {
		return fmt.Sprintf("%dm %ds", seconds/secondsPerMinute, seconds%secondsPerMinute)
	}
	return fmt.Sprintf("%dh %dm", seconds/secondsPerHour, (seconds%secondsPerHour)/secondsPerMinute)
}

const (
	secondsPerMinute = 60
	secondsPerHour   = 3600
)
