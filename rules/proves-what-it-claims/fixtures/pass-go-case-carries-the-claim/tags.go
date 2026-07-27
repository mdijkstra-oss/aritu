package scenario

import "strings"

func Normalize(raw string) []string {
	var normalized []string
	seen := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		tag := strings.TrimSpace(part)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		normalized = append(normalized, tag)
	}
	return normalized
}
