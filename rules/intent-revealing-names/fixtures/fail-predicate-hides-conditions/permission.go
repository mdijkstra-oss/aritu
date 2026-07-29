package access

import "time"

func IsGranted(role string, scopes []string, expiry time.Time, now time.Time) bool {
	if role != "admin" && role != "owner" {
		return false
	}
	if expiry.Before(now) || expiry.Sub(now) < time.Minute {
		return false
	}
	for _, scope := range scopes {
		if scope == "write" && role == "owner" {
			return true
		}
	}
	return false
}
