package scenario

import "strings"

type Normalizer interface {
	Normalize(address string) string
}

type EmailNormalizer struct{}

func (EmailNormalizer) Normalize(address string) string {
	trimmed := strings.TrimSpace(address)
	at := strings.LastIndex(trimmed, "@")
	if at < 0 {
		return strings.ToLower(trimmed)
	}
	return trimmed[:at] + "@" + strings.ToLower(trimmed[at+1:])
}
