package scenario

import (
	"strings"
	"unicode"
)

func Slugify(title string) string {
	var slug strings.Builder
	lastWasDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			slug.WriteRune(r)
			lastWasDash = false
		case !lastWasDash && slug.Len() > 0:
			slug.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.TrimSuffix(slug.String(), "-")
}
