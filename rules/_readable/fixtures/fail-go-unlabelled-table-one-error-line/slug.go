package scenario

import "strings"

func Slugify(title string) string {
	words := strings.FieldsFunc(title, isSeparator)
	lowered := make([]string, len(words))
	for i, word := range words {
		lowered[i] = strings.ToLower(word)
	}
	return strings.Join(lowered, "-")
}

func isSeparator(r rune) bool {
	return !isSlugRune(r)
}

func isSlugRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
