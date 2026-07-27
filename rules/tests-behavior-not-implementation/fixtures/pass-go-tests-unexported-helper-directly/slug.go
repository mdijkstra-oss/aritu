package scenario

import (
	"strings"
	"unicode"
)

func Slugify(title string) string {
	return collapseSeparators(strings.ToLower(title))
}

func collapseSeparators(s string) string {
	return strings.Join(strings.FieldsFunc(s, isSeparator), "-")
}

func isSeparator(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
