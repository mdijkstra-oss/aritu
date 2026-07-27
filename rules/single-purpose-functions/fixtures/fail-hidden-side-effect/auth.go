package auth

import "time"

func CheckPassword(user string, password string, sessions map[string]time.Time) bool {
	if len(password) < 8 {
		return false
	}
	sessions[user] = time.Now()
	return true
}
