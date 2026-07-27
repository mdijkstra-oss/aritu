package schedule

import "time"

func IsWeekend(day time.Weekday) bool {
	switch day {
	case time.Saturday, time.Sunday:
		return true
	case time.Monday, time.Tuesday, time.Wednesday, time.Thursday:
		return false
	default:
		return false
	}
}
