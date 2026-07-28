package lint

import (
	"fmt"
	"strings"
	"unicode"
)

// UnitsFor derives the key each enumerated identifier is answered under.
func UnitsFor(names []string) []Unit {
	units := make([]Unit, 0, len(names))
	for at, name := range names {
		units = append(units, Unit{Name: name, Key: keyFor(at, name)})
	}
	return units
}

// keyFor derives the property a unit answers under: its position in the listed
// units, then a normalised form of the name a reader can recognise.
//
// Uniqueness rides entirely on the position, which is what lets the readable half
// be cut to fit the API's ceiling. Cutting a readable key on its own is the wrong
// answer twice over: the prefix that survives is neither unique — two files under
// one long directory reduce to the same string — nor legible, and dropping a unit's
// own property would hand it a neighbour's verdict with every count still looking
// healthy. The tail is kept over the head because it is the half a reader
// recognises: the file name, the case that failed.
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

// snakeCase normalises a name for the key: anything outside the key's character
// set collapses to a single underscore however much of it there was, and a word
// break opens where a capital follows a lower-case letter or digit. Acronym-grade
// word splitting is deliberately absent — the position prefix already guarantees
// uniqueness, so the readable half only has to be recognisable.
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
