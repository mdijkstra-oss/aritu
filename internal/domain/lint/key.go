package lint

import (
	"fmt"
	"strings"
	"unicode"
)

func UnitsFor(names []string) []Unit {
	units := make([]Unit, 0, len(names))
	for at, name := range names {
		units = append(units, Unit{Name: name, Key: keyFor(at, name)})
	}
	return units
}

func keyFor(at int, name string) string {
	prefix := fmt.Sprintf("u%02d", at+1)
	readable := strings.Trim(lastChars(snakeCase(name), maxKeyLength-len(prefix)-1), "_")
	if readable == "" {
		return prefix
	}
	return prefix + "_" + readable
}

// maxKeyLength and the character set below are the API's, not ours: a schema
// property key is rejected outright unless it matches ^[a-zA-Z0-9_.-]{1,64}$.
// Colons, slashes and spaces are all out, which is why a unit's key cannot simply
// be the identifier a reader sees.
const maxKeyLength = 64

func lastChars(text string, count int) string {
	if len(text) <= count {
		return text
	}
	return text[len(text)-count:]
}

func snakeCase(text string) string {
	var b strings.Builder
	pendingSeparator := false
	var previous rune
	for _, r := range text {
		if !isWordRune(r) {
			pendingSeparator = true
			continue
		}
		opensWord := unicode.IsUpper(r) && (unicode.IsLower(previous) || unicode.IsDigit(previous))
		if b.Len() > 0 && (pendingSeparator || opensWord) {
			b.WriteByte('_')
		}
		pendingSeparator = false
		previous = r
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// isWordRune is deliberately ASCII: a rune outside this set becomes a separator
// rather than being lowercased into the key, which is what guarantees every key
// matches the character set the API accepts.
func isWordRune(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
}
