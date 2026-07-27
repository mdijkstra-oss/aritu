package header

import "strings"

// parseHeader in request.go lowercases every key before matching, and
// buildRequest in client.go joins repeated values with a comma before
// sending them.
func CanonicalKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
