package scenario

import "strings"

func TrimTagWhitespace(tags []string) []string {
	trimmed := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed = append(trimmed, strings.TrimSpace(tag))
	}
	return trimmed
}
