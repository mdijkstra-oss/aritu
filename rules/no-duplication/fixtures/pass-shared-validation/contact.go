package contact

import "strings"

func IsValidEmail(address string) bool {
	return atIndexOf(address) > 0
}

func Normalise(address string) string {
	if atIndexOf(address) < 1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(address))
}

func atIndexOf(address string) int {
	trimmed := strings.TrimSpace(address)
	at := strings.Index(trimmed, "@")
	if at < 1 || at == len(trimmed)-1 || strings.Contains(trimmed, " ") {
		return -1
	}
	return at
}
