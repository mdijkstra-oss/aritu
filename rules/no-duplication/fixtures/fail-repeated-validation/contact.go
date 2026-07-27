package contact

import "strings"

func IsValidEmail(address string) bool {
	trimmed := strings.TrimSpace(address)
	at := strings.Index(trimmed, "@")
	return at > 0 && at < len(trimmed)-1 && !strings.Contains(trimmed, " ")
}

func Normalise(address string) string {
	trimmed := strings.TrimSpace(address)
	at := strings.Index(trimmed, "@")
	if at > 0 && at < len(trimmed)-1 && !strings.Contains(trimmed, " ") {
		return strings.ToLower(trimmed)
	}
	return ""
}
