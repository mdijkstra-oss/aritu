package slug

import "strings"

func Make(title string) string {
	var b strings.Builder
	lastWasDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastWasDash = false
		case !lastWasDash && b.Len() > 0:
			b.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}
