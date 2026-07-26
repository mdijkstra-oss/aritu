package scenario

import "strings"

func NormalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		normalized = append(normalized, strings.ToLower(tag))
	}
	return normalized
}
