package scenario

import (
	"fmt"
	"slices"
	"strings"
)

type Index struct {
	entries map[string]string
	journal []string
}

func New() *Index {
	return &Index{entries: map[string]string{}}
}

func (i *Index) Add(title, target string) string {
	key := normalizeKey(title)
	i.entries[key] = target
	i.journal = append(i.journal, fmt.Sprintf("add %s -> %s", key, target))
	return key
}

func (i *Index) Lookup(key string) (string, bool) {
	target, found := i.entries[key]
	return target, found
}

func (i *Index) Journal() []string {
	return slices.Clone(i.journal)
}

func normalizeKey(title string) string {
	var key strings.Builder
	pendingDash := false
	for _, r := range strings.ToLower(title) {
		if !isAlphanumeric(r) {
			pendingDash = key.Len() > 0
			continue
		}
		if pendingDash {
			key.WriteRune('-')
			pendingDash = false
		}
		key.WriteRune(r)
	}
	return key.String()
}

func isAlphanumeric(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
}
